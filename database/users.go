package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sync"

	"github.com/Grarak/GoYTFetcher/utils"
	"golang.org/x/crypto/pbkdf2"
)

const TableUsers = "users"

type User struct {
	ApiKey       string   `json:"apikey,omitempty"`
	Name         string   `json:"name,omitempty"`
	Password     string   `json:"password,omitempty"`
	PasswordSalt string   `json:"-"`
	PasswordHash string   `json:"-"`
	Admin        *bool    `json:"admin,omitempty"`
	Verified     *bool    `json:"verified,omitempty"`
}

func NewUser(data []byte) (User, error) {
	var user User
	err := json.Unmarshal(data, &user)
	if err == nil {
		if user.Admin == nil {
			admin := false
			user.Admin = &admin
		}

		if user.Verified == nil {
			verified := false
			user.Verified = &verified
		}
	}
	return user, err
}

func (user User) ToJson() (string, error) {
	b, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func hashPassword(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, 4096, sha256.Size, sha256.New)
}

func generatePassword(password []byte) (string, string) {
	salt := utils.GenerateRandom(32)
	hash := hashPassword(password, salt)
	return utils.ToURLBase64(hash), utils.ToURLBase64(salt)
}

type UsersDB struct {
	db     *sql.DB
	rwLock *sync.RWMutex

	namePattern *regexp.Regexp
}

func newUsersDB(db *sql.DB, rwLock *sync.RWMutex) (*UsersDB, error) {
	cmd := newTableBuilder(TableUsers).
		addUniqueKeyPair(ColumnApikey).
		addUniqueKeyPair(ColumnName).
		addColumn(ColumnPasswordSalt).
		addColumn(ColumnPasswordHash).
		addColumn(ColumnAdmin).
		addColumn(ColumnVerified).build()

	_, err := db.Exec(cmd)
	if err != nil {
		return nil, err
	}

	regex, err := regexp.Compile("^[a-zA-Z0-9_]*$")
	if err != nil {
		return nil, err
	}

	return &UsersDB{db, rwLock, regex}, nil
}

func (usersDB *UsersDB) AddUser(user User) (User, int) {
	if len(user.Name) <= 3 {
		return user, utils.StatusNameShort
	}

	if len(user.Name) > 50 {
		return user, utils.StatusNameLong
	}

	if !usersDB.namePattern.MatchString(user.Name) {
		return user, utils.StatusNameInvalid
	}

	password, err := utils.Decode(user.Password)
	if err != nil {
		return user, utils.StatusPasswordInvalid
	}

	if len(password) <= 4 {
		return user, utils.StatusPasswordShort
	}

	if len(password) > 50 {
		return user, utils.StatusPasswordLong
	}

	usersDB.rwLock.Lock()
	defer usersDB.rwLock.Unlock()

	if _, err := usersDB.findUserByName(user.Name); err == nil {
		return user, utils.StatusUserAlreadyExists
	}

	// Hash password
	hash, salt := generatePassword(password)

	// Generate api token
	user.ApiKey = usersDB.generateApiToken()

	user.Password = ""

	// If this is the first user
	// Make him admin
	count, _ := rowCountInTable(usersDB.db, TableUsers)
	var admin bool
	var verified bool
	if count == 0 {
		admin = true
		verified = true
	}
	user.Admin = &admin
	user.Verified = &verified

	_, err = usersDB.db.Exec(fmt.Sprintf(

		"INSERT INTO %s "+
			"(%s, %s, %s, %s, %s, %s) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		TableUsers,
		ColumnApikey.name, ColumnName.name,
		ColumnPasswordSalt.name, ColumnPasswordHash.name,
		ColumnAdmin.name, ColumnVerified.name),

		user.ApiKey, user.Name, salt, hash,
		*user.Admin, *user.Verified)
	if err != nil {
		return user, utils.StatusAddUserFailed
	}

	return user, utils.StatusNoError
}

func (usersDB *UsersDB) GetUserWithPassword(name, password string) (User, int) {
	user, err := usersDB.FindUserByName(name)
	if err == nil {
		password, err := utils.Decode(password)
		if err == nil {
			salt, err := utils.FromURLBase64(user.PasswordSalt)
			if err == nil {
				newHash := hashPassword(password, salt)
				oldHash, err := utils.FromURLBase64(user.PasswordHash)
				if err == nil && reflect.DeepEqual(oldHash, newHash) {
					user.Password = ""
					return user, utils.StatusNoError
				}
			}
		}
	}

	return User{}, utils.StatusInvalidPassword
}

func (usersDB *UsersDB) generateApiToken() string {
	token := utils.ToURLBase64(utils.GenerateRandom(32))
	if _, err := usersDB.findUserByApiKey(token); err == nil {
		return usersDB.generateApiToken()
	}
	return token
}

func (usersDB *UsersDB) FindUserByApiKey(apiKey string) (User, error) {
	usersDB.rwLock.RLock()
	defer usersDB.rwLock.RUnlock()
	return usersDB.findUserByApiKey(apiKey)
}

func (usersDB *UsersDB) findUserByApiKey(apiKey string) (User, error) {
	users, err := usersDB.createUserWithWhere(
		ColumnApikey.name+" = ?", apiKey)
	if len(users) > 0 {
		return users[0], err
	}
	return User{}, fmt.Errorf("no users found")
}

func (usersDB *UsersDB) FindUserByName(name string) (User, error) {
	usersDB.rwLock.RLock()
	defer usersDB.rwLock.RUnlock()
	return usersDB.findUserByName(name)
}

func (usersDB *UsersDB) findUserByName(name string) (User, error) {
	users, err := usersDB.createUserWithWhere(
		ColumnName.name+" = ? COLLATE NOCASE", name)
	if len(users) > 0 {
		return users[0], err
	}
	return User{}, fmt.Errorf("no users found")
}

func (usersDB *UsersDB) ListUsers(page int) ([]User, error) {
	usersDB.rwLock.RLock()
	defer usersDB.rwLock.RUnlock()

	if page < 1 {
		page = 1
	}
	users, err := usersDB.createUsers(fmt.Sprintf(
		"LIMIT 10 OFFSET %d", 10*(page-1)))
	if err != nil {
		return nil, err
	}

	usersNoApiKey := make([]User, len(users))
	for i := range users {
		usersNoApiKey[i] = users[i]
		usersNoApiKey[i].ApiKey = ""
	}
	return usersNoApiKey, nil
}

func (usersDB *UsersDB) SetVerificationUser(request User) error {
	usersDB.rwLock.Lock()
	defer usersDB.rwLock.Unlock()

	_, err := usersDB.db.Exec(fmt.Sprintf(
		"UPDATE %s SET %s = ? WHERE %s = ?",
		TableUsers, ColumnVerified.name, ColumnName.name), *request.Verified, request.Name)
	return err
}

func (usersDB *UsersDB) DeleteUser(request User) error {
	usersDB.rwLock.Lock()
	defer usersDB.rwLock.Unlock()

	_, err := usersDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = ?",
		TableUsers, ColumnName.name), request.Name)
	return err
}

func (usersDB *UsersDB) DeleteAllNonVerifiedUsers(request User) error {
	usersDB.rwLock.Lock()
	defer usersDB.rwLock.Unlock()

	_, err := usersDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = %d OR %s = null",
		TableUsers, ColumnVerified.name, 0, ColumnVerified.name))
	return err
}

func (usersDB *UsersDB) ResetPasswordUser(request User) error {
	usersDB.rwLock.Lock()
	defer usersDB.rwLock.Unlock()

	password, err := utils.Decode(request.Password)
	if len(password) <= 4 {
		return fmt.Errorf("password too short")
	}

	if err != nil {
		return err
	}
	hash, salt := generatePassword(password)
	_, err = usersDB.db.Exec(fmt.Sprintf(
		"UPDATE %s SET %s = ?, %s = ? WHERE %s = ?",
		TableUsers, ColumnPasswordHash.name,
		ColumnPasswordSalt.name,
		ColumnName.name), hash, salt, salt)
	return err
}

func (usersDB *UsersDB) createUserWithWhere(where string, args ...interface{}) ([]User, error) {
	return usersDB.createUsers("WHERE "+where, args...)
}

func (usersDB *UsersDB) createUsers(condition string, args ...interface{}) ([]User, error) {
	stmt, err := usersDB.db.Prepare(fmt.Sprintf(
		"SELECT %s,%s,%s,%s,%s,%s FROM %s %s",
		ColumnApikey.name, ColumnName.name, ColumnPasswordSalt.name,
		ColumnPasswordHash.name, ColumnAdmin.name,
		ColumnVerified.name, TableUsers, condition))
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		admin := false
		verified := false
		user := User{Admin: &admin, Verified: &verified}
		err := rows.Scan(&user.ApiKey, &user.Name, &user.PasswordSalt,
			&user.PasswordHash, user.Admin, user.Verified)
		if err != nil {
			return nil, err
		}
		if utils.StringIsEmpty(user.Name) {
			return nil, fmt.Errorf("couldn't find user with %s", condition)
		}
		users = append(users, user)
	}
	return users, nil
}
