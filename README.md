# faceserver
人脸特征提取服务器版本  
需要通过websocket连接服务器，按照指定格式提交请求。  

# 配置  
windows 平台  
将第三方库的lib文件放在 xface目录下  
动态库放在build目录下，model文件夹放在build目录下  

linux平台  
将第三方库放在build目录下，model文件夹放在build目录下  

#命令行  
服务器侦听：  
./faceserver --listen=:9979 -v=4 -alsologtostderr  
后面两项可选，可以参考 google glog命令行  

shell命令行：  
./faceserver --cmd=stop //停止faceserver  
./faceserver --version //查看app版本  
