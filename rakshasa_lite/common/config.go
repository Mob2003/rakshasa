package common

type Config struct {
	UUID     string   //以指定uuid启动
	DstNode  []string //-d 上级节点
	Password string   //通讯密码,可为空
	Port     int      //默认8883
	ListenIp string //指定公网ip，其他节点进行额外节点连接时候，尝试连接的ip
	Limit    bool     //禁止额外连接,只连接-d节点,不会尝试连接其他节点
	FileName string
	FileSave bool `yaml:"-"`
}
