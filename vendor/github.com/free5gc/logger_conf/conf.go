package logger_conf

import (
	"log"
	"os"
	"strconv"

	"github.com/free5gc/path_util"
)

var Free5gcLogDir string = path_util.Free5gcPath("free5gc/log") + "/"
var LibLogDir string = Free5gcLogDir + "lib/"
var NfLogDir string = Free5gcLogDir + "nf/"

var Free5gcLogFile string = Free5gcLogDir + "free5gc.log"

func init() {
	if err := os.MkdirAll(LibLogDir, 0775); err != nil {
		log.Printf("Mkdir %s failed: %+v", LibLogDir, err)
	}
	if err := os.MkdirAll(NfLogDir, 0775); err != nil {
		log.Printf("Mkdir %s failed: %+v", NfLogDir, err)
	}

	// Create log file or if it already exist, check if user can access it
	f, fileOpenErr := os.OpenFile(Free5gcLogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if fileOpenErr != nil {
		// user cannot access it.
		log.Printf("Cannot Open %s\n", Free5gcLogFile)
	} else {
		// user can access it
		if err := f.Close(); err != nil {
			log.Printf("File %s cannot been closed\n", Free5gcLogFile)
		}
	}

	sudoUID, errUID := strconv.Atoi(os.Getenv("SUDO_UID"))
	sudoGID, errGID := strconv.Atoi(os.Getenv("SUDO_GID"))

	if errUID == nil && errGID == nil {
		// if using sudo to run the program, errUID will be nil and sudoUID will get the uid who run sudo
		// else errUID will not be nil and sudoUID will be nil
		// If user using sudo to run the program and create log file, log will own by root,
		// here we change own to user so user can view and reuse the file
		if err := os.Chown(Free5gcLogDir, sudoUID, sudoGID); err != nil {
			log.Printf("Dir %s chown to %d:%d error: %v\n", Free5gcLogDir, sudoUID, sudoGID, err)
		}
		if err := os.Chown(LibLogDir, sudoUID, sudoGID); err != nil {
			log.Printf("Dir %s chown to %d:%d error: %v\n", LibLogDir, sudoUID, sudoGID, err)
		}
		if err := os.Chown(NfLogDir, sudoUID, sudoGID); err != nil {
			log.Printf("Dir %s chown to %d:%d error: %v\n", NfLogDir, sudoUID, sudoGID, err)
		}

		if fileOpenErr == nil {
			if err := os.Chown(Free5gcLogFile, sudoUID, sudoGID); err != nil {
				log.Printf("File %s chown to %d:%d error: %v\n", Free5gcLogFile, sudoUID, sudoGID, err)
			}
		}
	}
}
