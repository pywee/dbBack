package main

/*
* @Author: pywee
* @Date: 2024-11-30 09:52:53
* @Last Modified time: 2024-11-30 09:52:53
* @Description: 数据库备份脚本
 */

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

var (
	// confFile 数据库备份信息
	confFile = ".backup"
	// mysqlBackupDir mysql 数据库备份目录
	mysqlBackupDir = "/mysqlBackup"
	// mongoBackupDir mongoDB 数据库备份目录
	mongoBackupDir = "/mongodbBackup"
	// mysqlPwd mysql 数据库密码 不同环境密码不同
	mysqlPwd = "MYSQL_PWD"
)

type Config struct {
	// DBType 数据库类型 [1.MySQL; 2.MongoDB]
	DBType uint8 `json:"dbType"`
	// Cycle 备份频率 单位：分钟
	Cycle int64 `json:"cycle"`
	// Path 备份路径
	Path string `json:"path"`
	// NextBackupTs 下一次备份时间戳
	NextBackupTs int64 `json:"nextBackupTs"`
	// NextBackupDate 下一次备份日期时间
	NextBackupDate string `json:"nextBackupDate"`
}

// 备份 MongoDB 和 MySQL 数据库
func main() {
	if env := os.Getenv("ENV"); env == "prod" {
		mysqlPwd = "MYSQL_PWD"
	}
	for {
		if err := doBackup(); err != nil {
			fmt.Println("备份错误：", err)
		}
		time.Sleep(time.Second * 30)
	}
}

func doBackup() error {
	var (
		conf []*Config
		tn   = time.Now()
	)

	b, err := os.ReadFile(confFile)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(os.TempDir(), 0755); err != nil {
			return err
		}
		f, err := os.Create(confFile)
		if err != nil {
			return err
		}
		f.Close()

		conf = append(conf, &Config{DBType: 1, Cycle: 1440, Path: mysqlBackupDir + "/1day.sql"})
		conf = append(conf, &Config{DBType: 1, Cycle: 10800, Path: mysqlBackupDir + "/7days.sql"})
		conf = append(conf, &Config{DBType: 1, Cycle: 43200, Path: mysqlBackupDir + "/30days.sql"})

		conf = append(conf, &Config{DBType: 2, Cycle: 1440, Path: mongoBackupDir + "/1day"})
		conf = append(conf, &Config{DBType: 2, Cycle: 10800, Path: mongoBackupDir + "/7days"})
		conf = append(conf, &Config{DBType: 2, Cycle: 43200, Path: mongoBackupDir + "/30days"})
	} else if err = json.Unmarshal(b, &conf); err != nil {
		return err
	}

	for k, v := range conf {
		thisCycle := 60 * v.Cycle
		if ts := tn.Unix(); ts >= v.NextBackupTs {
			if v.DBType == 1 {
				if err = doMysqlBackupHandler(v); err != nil {
					fmt.Printf("fail to backup mysql, path: %s, err: %v\n", v.Path, err)
					continue
				}
				fmt.Printf("success, path: %s\n", v.Path)
			} else if v.DBType == 2 {
				if err = doMongodbBackupHandler(v); err != nil {
					fmt.Printf("fail to backup mongodb, path: %s, err: %v\n", v.Path, err)
					continue
				}
				fmt.Printf("success, path: %s\n", v.Path)
			}
			nts := ts + thisCycle
			conf[k].NextBackupTs = nts
			conf[k].NextBackupDate = time.Unix(nts, 0).Format("2006-01-02 15:04:05")
		}
	}

	nb, _ := json.MarshalIndent(conf, "", "  ")
	os.WriteFile(confFile, nb, 0644)

	return nil
}

// docker exec -t mysql mysqldump -u root -pMYSQL_PWD 数据库名称 > /mysqlBackup/backup.sql
func doMysqlBackupHandler(conf *Config) error {
	fmt.Println("ready to backup mysql...")
	cmd := exec.Command(
		"docker",
		"exec",
		"-t",
		"mysql",
		"mysqldump",
		"-u", "root",
		"-p"+mysqlPwd,
		"数据库名称",
	)

	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return compressToZip(outputBuffer.Bytes(), conf.Path+".zip")
}

// docker exec -t mongodbv604 mongodump --host 127.0.0.1 --port 27017 --username root --password MONGO_PWD --authenticationDatabase admin --out /mongodbBackup/1day
func doMongodbBackupHandler(conf *Config) error {
	fmt.Println("ready to backup mongodb...")

	cmd := exec.Command(
		"docker", "exec", "-t", "mongodbv604",
		"mongodump",
		"--host", "127.0.0.1",
		"--port", "27017",
		"--username", "root",
		"--password", "MONGO_PWD",
		"--authenticationDatabase", "admin",
		"--archive="+conf.Path+".gz",
		"--gzip",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func compressToZip(data []byte, zipFilePath string) error {
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	zipEntry, err := zipWriter.Create("data")
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	if _, err := zipEntry.Write(data); err != nil {
		return fmt.Errorf("failed to write data to zip: %w", err)
	}

	return nil
}
