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

# 编译  
因为用到了cgo，所以：   
windows 下需要安装mingw64  
设置环境变量：  
CGO_ENABLED=1
GOPROXY=https://goproxy.cn
PATH中添加： D:\GNU\msys64\mingw64\bin


linux 下需要安装 gcc  
设置代理： go env -w GOPROXY=https://goproxy.cn,direct  

