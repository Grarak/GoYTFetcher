package logger

import (
	"runtime"
	"fmt"
	"time"
)

func I(message interface{}) {
	fmt.Println(getTime()+": "+getFunctionName()+":", message)
}

func E(message interface{}) {
	fmt.Println(getTime()+": "+getFunctionName()+":", message)
}

func getTime() string {
	return time.Now().Format(time.StampMilli)
}

func getFunctionName() string {
	fpcs := make([]uintptr, 1)

	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return "n/a"
	}

	fun := runtime.FuncForPC(fpcs[0])
	if fun == nil {
		return "n/a"
	}

	return fun.Name()
}
