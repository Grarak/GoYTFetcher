package database

import (
	"database/sql"
	"../utils"

	_ "github.com/mattn/go-sqlite3"

	"regexp"
	"fmt"
	"reflect"
	"encoding/json"
	"golang.org/x/crypto/pbkdf2"
	"crypto/sha256"
	"sync"
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
	Playlist     []string `json:"playlist,omitempty"`
	History      []string `json:"history,omitempty"`
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

type UserDB struct {
	db     *sql.DB
	rwLock *sync.RWMutex

	namePattern *regexp.Regexp
}

func newUserDB(db *sql.DB, rwLock *sync.RWMutex) (*UserDB, error) {
	err := createTablesWithPrimaryKeys(db, []column{ColumnApikey, ColumnName}, TableUsers)
	if err != nil {
		return nil, err
	}

	err = insertRows(db, TableUsers, ColumnPasswordSalt,
		ColumnPasswordHash, ColumnAdmin, ColumnVerified)
	if err != nil {
		return nil, err
	}

	regex, err := regexp.Compile("^[a-zA-Z0-9_]*$")
	if err != nil {
		return nil, err
	}

	return &UserDB{db, rwLock, regex}, nil
}

func (userDB *UserDB) AddUser(user User) (User, int) {
	if len(user.Name) <= 3 {
		return user, utils.StatusNameShort
	}

	if len(user.Name) > 50 {
		return user, utils.StatusNameLong
	}

	if !userDB.namePattern.MatchString(user.Name) {
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

	userDB.rwLock.Lock()
	defer userDB.rwLock.Unlock()

	if _, err := userDB._findUserByName(user.Name, false); err == nil {
		return user, utils.StatusUserAlreadyExists
	}

	// Hash password
	hash, salt := generatePassword(password)

	// Generate api token
	user.ApiKey = userDB.generateApiToken()

	user.Password = ""

	// If this is the first user
	// Make him admin
	count, _ := rowCountInTable(userDB.db, TableUsers)
	var admin bool
	var verified bool
	if count == 0 {
		admin = true
		verified = true
	}
	user.Admin = &admin
	user.Verified = &verified

	_, err = userDB.db.Exec(fmt.Sprintf(

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

func (userDB *UserDB) GetUserWithPassword(name, password string) (User, int) {
	user, err := userDB.FindUserByName(name)
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

func (userDB *UserDB) generateApiToken() string {
	token := utils.ToURLBase64(utils.GenerateRandom(32))
	if _, err := userDB.FindUserByApiKey(token); err == nil {
		return userDB.generateApiToken()
	}
	return token
}

func (userDB *UserDB) FindUserByApiKey(apiKey string) (User, error) {
	userDB.rwLock.RLock()
	defer userDB.rwLock.RUnlock()

	users, err := userDB.createUserWithWhere(
		ColumnApikey.name + " = " + "'" + apiKey + "'")
	if len(users) > 0 {
		return users[0], err
	}
	return User{}, utils.Error("No users found!")
}

func (userDB *UserDB) FindUserByName(name string) (User, error) {
	return userDB._findUserByName(name, true)
}

func (userDB *UserDB) _findUserByName(name string, lock bool) (User, error) {
	if lock {
		userDB.rwLock.RLock()
		defer userDB.rwLock.RUnlock()
	}

	users, err := userDB.createUserWithWhere(
		ColumnName.name + " = " + "'" + name + "' COLLATE NOCASE")
	if len(users) > 0 {
		return users[0], err
	}
	return User{}, utils.Error("No users found!")
}

func (userDB *UserDB) ListUsers(page int) ([]User, error) {
	userDB.rwLock.RLock()
	defer userDB.rwLock.RUnlock()

	if page < 1 {
		page = 1
	}
	return userDB.createUsers(fmt.Sprintf(
		"LIMIT 10 OFFSET %d", 10*(page-1)))
}

func (userDB *UserDB) SetVerificationUser(request User) error {
	userDB.rwLock.Lock()
	defer userDB.rwLock.Unlock()

	var verified int
	if *request.Verified {
		verified = 1
	}
	_, err := userDB.db.Exec(fmt.Sprintf(
		"UPDATE %s SET %s = %d WHERE %s = '%s'",
		TableUsers, ColumnVerified.name, verified, ColumnName.name, request.Name))
	return err
}

func (userDB *UserDB) DeleteUser(request User) error {
	userDB.rwLock.Lock()
	defer userDB.rwLock.Unlock()

	_, err := userDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = '%s'",
		TableUsers, ColumnName.name, request.Name))
	return err
}

func (userDB *UserDB) DeleteAllNonVerifiedUsers(request User) error {
	userDB.rwLock.Lock()
	defer userDB.rwLock.Unlock()

	_, err := userDB.db.Exec(fmt.Sprintf(
		"DELETE FROM %s WHERE %s = %d OR %s = null",
		TableUsers, ColumnVerified.name, 0, ColumnVerified.name))
	return err
}

func (userDB *UserDB) ResetPasswordUser(request User) error {
	userDB.rwLock.Lock()
	defer userDB.rwLock.Unlock()

	password, err := utils.Decode(request.Password)
	if len(password) <= 4 {
		return utils.Error("Password too short")
	}

	if err != nil {
		return err
	}
	hash, salt := generatePassword(password)
	_, err = userDB.db.Exec(fmt.Sprintf(
		"UPDATE %s SET %s = '%s', %s = '%s' WHERE %s = '%s'",
		TableUsers, ColumnPasswordHash.name, hash,
		ColumnPasswordSalt.name, salt,
		ColumnName.name, request.Name))
	return err
}

func (userDB *UserDB) createUserWithWhere(where string) ([]User, error) {
	return userDB.createUsers("WHERE " + where)
}

func (userDB *UserDB) createUsers(condition string) ([]User, error) {
	cmd := fmt.Sprintf(
		"SELECT %s,%s,%s,%s,%s,%s FROM %s %s",
		ColumnApikey.name, ColumnName.name, ColumnPasswordSalt.name,
		ColumnPasswordHash.name, ColumnAdmin.name,
		ColumnVerified.name,

		TableUsers, condition)

	row, err := userDB.db.Query(cmd)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var users []User
	for row.Next() {
		admin := false
		verified := false
		user := User{Admin: &admin, Verified: &verified}
		err := row.Scan(&user.ApiKey, &user.Name, &user.PasswordSalt,
			&user.PasswordHash, user.Admin, user.Verified)
		if err != nil {
			return nil, err
		}
		if utils.StringIsEmpty(user.Name) {
			return nil, utils.Error("Couldn't find user with " + condition)
		}
		users = append(users, user)
	}
	return users, nil
}
