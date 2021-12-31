package recover

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

func PanicHandler() {
	if err := recover(); err != nil {
		exeName := filepath.Base(os.Args[0])

		now := time.Now()
		pid := os.Getpid()

		time_str := now.Format("20060102150405")
		fname := fmt.Sprintf("/log/xiot/debug/%s-%d-%s-dump.log", exeName, pid, time_str)
		fmt.Println("dump to file ", fname)

		f, errCreate := os.Create(fname)
		if errCreate != nil {
			fmt.Println(errCreate)
			return
		}
		f.WriteString(fmt.Sprintf("%v\r\n", err))
		f.WriteString("========\r\n")
		f.WriteString(string(debug.Stack()))
		f.Close()
	}
}

