package log4g

import (
	"os"
	"path/filepath"
	"encoding/json"
	"log"
	"io/ioutil"
)
type LogConfig struct {
	LogToFile	bool
	LogOutPath string
}
var logfile *log.Logger
var logConfig LogConfig
func LoadConfig(configPath string)  {
	var (
		err error
		data []byte
		file *os.File
	)
	defer file.Close()
	rootPath,_ := os.Getwd()
	path := filepath.Join(rootPath,configPath)
	if _,err = os.Stat(path);err != nil{
		//不存在创建
		logConfig = LogConfig{
			LogToFile:false,
			LogOutPath:"/output.log",
		}
		data,err=json.Marshal(logConfig)
		if err != nil{
			panic(err)
		}
		file,err = os.Create(path)
		if err != nil{
			panic(err)
		}
		_,err = file.Write(data)
		if err != nil{
			panic(err)
		}
	}else{
		//读取
		file,err = os.Open(path)
		if err != nil{
			panic(err)
		}
		data,err = ioutil.ReadAll(file)
		if err != nil{
			panic(err)
		}
		logConfig = LogConfig{}
		json.Unmarshal(data,logConfig)
	}
	if &logConfig != nil{
		if logConfig.LogToFile{
			if logfile == nil{
				path = filepath.Join(rootPath,logConfig.LogOutPath)
				file,err := os.Create(path)
				if err != nil{
					panic(err)
				}
				logfile = log.New(file,"[Info]",log.LstdFlags)
			}
		}
	}
}
func Info(content string)  {
	if logfile != nil{
		logfile.SetPrefix("[Info]")
		logfile.Println(content)
	}else{
		log.Println(content)
	}
}
func Error(content string){
	if logfile != nil{
		logfile.SetPrefix("[Error]")
		logfile.Panicln(content)
	}else{
		log.Panicln(content)
	}
}