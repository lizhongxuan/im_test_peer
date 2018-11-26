package main

import (
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"./config"
	"sync/atomic"
	"time"
)

var sendNum int64
var receiveNum int64

var (
	cfg = pflag.StringP("config", "c", "", "apiserver config file path.")
)

func main() {

	glog.Infof("begin run im_help \n")

	pflag.Parse()
	// init config
	if err := config.Init(*cfg); err != nil {
		panic(err)
	}
	glog.Info("config.im_address:", viper.GetString("im_address"))
	glog.Info("config.msg_content:", viper.GetString("msg_content"))
	glog.Info("config.sec_send_msg:", viper.GetInt64("sec_send_msg"))
	glog.Info("config.uidNum:", viper.GetInt64("uidNum"))
	glog.Info("config.head_id:", viper.GetInt64("head_id"))
	head_id := viper.GetInt64("head_id")

	h := head_id * 100000000000

	var group_start int64 = viper.GetInt64("group_start")
	var startUid int64
	startUid = h + group_start*100000
	glog.Info("startUid:", startUid)

	go creadGroupTest(viper.GetString("im_address"), startUid, viper.GetInt64("uidNum"), viper.GetString("msg_content"))
	select {}
}

func (client *Channel) sendMsg(seq int, uid int64, receiver int64, content string) {
	glog.Info("sendMsg --- receiver :", receiver, "   send :", client.uid)
	msg := &Message{MSG_IM, seq, DEFAULT_VERSION, 0, &IMMessage{uid, receiver, 0, int32(1), content}}
	client.wt <- msg
	atomic.AddInt64(&sendNum, 1)
	atomic.AddInt64(&client.sendM, 1)
}

func creadGroupTest(addr string, startUid int64, member int64, content string) {
	defer glog.Info("creadGroupTest end")
	var i int64
	var clientMater *Channel
	for ; i < member; i++ {
		if i == 0 {
			clientMater = NewChannel(addr, startUid+i)
			go clientMater.Start()
			glog.Info("cread clientMater:", clientMater.uid)
			continue
		}
		if i%50 == 0 {
			time.Sleep(100 * time.Millisecond)
		}

		client := NewChannel(addr, startUid+i)
		go client.Start()
		glog.Info("cread client:", client.uid)
	}

	time.Sleep(3 * time.Second) //启动时间
	atomic.StoreInt64(&sendNum, 0)
	atomic.StoreInt64(&receiveNum, 0)
	for {
		ssm := viper.GetInt64("sec_send_msg")

		for ssm != 0 {
			var j int64=1
			for ;j<member ;j++  {
				clientMater.sendMsg(1, startUid, startUid+j, content)
				glog.Info("all ---- sendNum:", sendNum, "  receiveNum:", receiveNum)
			}
			ssm = viper.GetInt64("sec_send_msg")
		}
		time.Sleep(time.Second)
	}


}
