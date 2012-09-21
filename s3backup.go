// This code is under BSD license. See license-bsd.txt
package main

/*
func upload(bucket s3.Bucket, local, remote string, public bool) error {
	localf, err := os.Open(local)
	if err != nil {
		return err
	}
	defer localf.Close()
	localfi, err := localf.Stat()
	if err != nil {
		return err
	}

	auth, region, err := readConfig()
	if err != nil {
		return err
	}

	var bucket, name string
	if i := strings.Index(remote, "/"); i >= 0 {
		bucket, name = remote[:i], remote[i+1:]
		if name == "" || strings.HasSuffix(name, "/") {
			name += path.Base(local)
		}
	} else {
		bucket = remote
		name = path.Base(local)
	}

	acl := s3.Private
	if public {
		acl = s3.PublicRead
	}

	contType := mime.TypeByExtension(path.Ext(local))
	if contType == "" {
		contType = "binary/octet-stream"
	}

	err = b.PutBucket(acl)
	if err != nil {
		return err
	}
	return b.PutReader(name, localf, localfi.Size(), contType, acl)
}
*/

import (
	"archive/zip"
	"fmt"
	"io"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"log"
	_ "mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var backupFreq = 4 * time.Hour
var bucketDelim = "/"

type BackupConfig struct {
	AwsAccess string
	AwsSecret string
	Bucket    string
	S3Dir     string
	LocalDir  string
}

func ensureValidConfig(config *BackupConfig) {
	if !PathExists(config.LocalDir) {
		log.Fatalf("Invalid s3 backup: directory to backup '%s' doesn't exist\n", config.LocalDir)
	}

	if !strings.HasSuffix(config.S3Dir, bucketDelim) {
		config.S3Dir += bucketDelim
	}

	auth := aws.Auth{config.AwsAccess, config.AwsSecret}
	s3 := s3.New(auth, aws.USEast)
	bucket := s3.Bucket(config.Bucket)
	_, err := bucket.List(config.S3Dir, bucketDelim, "", 10)
	if err != nil {
		log.Fatalf("Invalid s3 backup: bucket.List failed %s\n", err.Error())
	}
	fmt.Printf("s3 bucket ok!\n")
}

func doBackup(config *BackupConfig) {
	// TODO: a better way to generate a random file name
	path := filepath.Join(os.TempDir(), "apptranslator-tmp-backup.zip")
	fmt.Printf("zip file name: %s\n", path)
	os.Remove(path)
	zf, err := os.Create(path)
	if err != nil {
		// TODO: what to do about it? Notify using e-mail?
		return
	}
	defer zf.Close()
	//defer os.Remove(path)
	zipWriter := zip.NewWriter(zf)
	// TODO: is the order of defer here can create problems?
	// TODO: need to check error code returned by Close()
	defer zipWriter.Close()

	//fmt.Printf("Walk root: %s\n", config.LocalDir)
	root := config.LocalDir
	rootLen := len(config.LocalDir) + 1 // +1 for slash
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//fmt.Printf("%s\n", path)
		if root == path {
			return nil
		}
		toZipPath := filepath.Join(root, path[rootLen:])
		fmt.Printf("toZipPath: %s\n", toZipPath)
		isDir, err := PathIsDir(toZipPath)
		if err != nil {
			return err
		}
		if isDir {
			return nil
		}
		toZipReader, err := os.Open(toZipPath)
		if err != nil {
			return err
		}
		defer toZipReader.Close()

		inZipWriter, err := zipWriter.Create(toZipPath)
		if err != nil {
			fmt.Printf("Error in zipWriter(): %s\n", err.Error())
			return err
		}
		_, err = io.Copy(inZipWriter, toZipReader)
		if err != nil {
			return err
		}
		fmt.Printf("Added %s to zip file\n", toZipPath)
		return nil
	})
	if err != nil {
		return
	}
}

func BackupLoop(config *BackupConfig) {
	ensureValidConfig(config)
	doBackup(config)
	log.Fatalf("Exiting now")
	for {
		// sleep first so that we don't backup right after new deploy
		time.Sleep(backupFreq)
		fmt.Printf("Doing backup to s3\n")
		//b := s3.New(auth, region).Bucket(bucket)
	}
}
