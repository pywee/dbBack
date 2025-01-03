


执行程序后，会在当前目录下生成一个名为 .backup 的配置文件，该文件作为执行计划，里面包含一些关键字段：

```golang
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

```
