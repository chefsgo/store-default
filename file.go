package file_default

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"

	. "github.com/chefsgo/base"
	"github.com/chefsgo/file"
	"github.com/chefsgo/util"
)

//-------------------- defaultBase begin -------------------------

type (
	defaultDriver  struct{}
	defaultConnect struct {
		mutex  sync.RWMutex
		health file.Health

		setting    defaultSetting
		sharedring *util.HashRing
	}
	defaultSetting struct {
		Storage  string
		Sharding int
	}
)

//连接
func (driver *defaultDriver) Connect(instance file.Instance) (file.Connect, error) {
	if config.Cache == "" {
		config.Cache = os.TempDir()
	}

	setting := defaultSetting{
		Sharding: 2000, Storage: "asset/storage",
	}

	//分片环
	weights := map[string]int{}
	for i := 1; i <= setting.Sharding; i++ {
		weights[fmt.Sprintf("%v", i)] = 1
	}

	return &defaultConnect{
		name: name, config: config, setting: setting,
		sharedring: util.NewHashRing(weights),
	}, nil

}

//打开连接
func (connect *defaultConnect) Open() error {
	return nil
}

func (connect *defaultConnect) Health() file.Health {
	connect.mutex.RLock()
	defer connect.mutex.RUnlock()
	return connect.health
}

//关闭连接
func (connect *defaultConnect) Close() error {
	return nil
}

func (connect *defaultConnect) Upload(target string, metadata Map) (file.File, file.Files, error) {
	stat, err := os.Stat(target)
	if err != nil {
		return nil, nil, err
	}

	//是目录
	if stat.IsDir() {

		dirs, err := ioutil.ReadDir(target)
		if err != nil {
			return nil, nil, err
		}

		files := file.Files{}
		for _, file := range dirs {
			if !file.IsDir() {

				source := path.Join(target, file.Name())
				hash := file.Hash(source)
				if hash == "" {
					return nil, nil, errors.New("hash error")
				}

				file := file.NewFile(connect.name, hash, source, file.Size())

				err := connect.storage(source, file)
				if err != nil {
					return nil, nil, err
				}

				files = append(files, file)
			}
		}

		return nil, files, nil

	} else {

		hash := file.Hash(target)
		if hash == "" {
			return nil, nil, errors.New("hash error")
		}

		file := file.NewFile(connect.name, hash, target, stat.Size())

		err := connect.storage(target, file)
		if err != nil {
			return nil, nil, err
		}

		return file, nil, nil
	}
}

func (connect *defaultConnect) Download(file file.File) (string, error) {
	///直接返回本地文件存储
	_, _, sFile, err := connect.storaging(file)
	if err != nil {
		return "", err
	}
	return sFile, nil
}

func (connect *defaultConnect) Remove(file file.File) error {
	_, _, sFile, err := connect.storaging(file)
	if err != nil {
		return err
	}

	return os.Remove(sFile)
}

// func (connect *defaultConnect) Browse(file file.File, name string, expiries ...time.Duration) (string, error) {
// 	return argo.Browse(file.Code(), name, expiries...), nil
// }

// func (connect *defaultConnect) Preview(file file.File, w, h, t int64, expiries ...time.Duration) (string, error) {
// 	return argo.Preview(file.Code(), w, h, t, expiries...), nil
// }

//-------------------- defaultBase end -------------------------

func (connect *defaultConnect) storage(source string, coding file.File) error {
	_, _, sFile, err := connect.storaging(coding)
	if err != nil {
		return err
	}

	//如果文件已经存在，直接返回
	if _, err := os.Stat(sFile); err == nil {
		return nil
	}

	//打开原始文件
	fff, err := os.Open(source)
	if err != nil {
		return err
	}
	defer fff.Close()

	//创建文件
	save, err := os.OpenFile(sFile, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	defer save.Close()

	//复制文件
	_, err = io.Copy(save, fff)
	if err != nil {
		return err
	}

	return nil
}

func (connect *defaultConnect) storaging(file file.File) (string, string, string, error) {
	if ring := connect.sharedring.Locate(file.Hash()); ring != "" {

		full := file.Hash()
		if file.Type() != "" {
			full = fmt.Sprintf("%s.%s", file.Hash(), file.Type())
		}

		spath := path.Join(connect.setting.Storage, ring)
		sfile := path.Join(spath, full)

		// //创建目录
		err := os.MkdirAll(spath, 0777)
		if err != nil {
			return "", "", "", errors.New("生成目录失败")
		}

		return ring, spath, sfile, nil
	}

	return "", "", "", errors.New("配置异常")
}
