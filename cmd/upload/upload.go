package upload

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var UploadCmd = &cobra.Command{
	Use:     "upload",
	Aliases: []string{"u"},
	Short:   `Upload file to pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		pikpak.Concurrent = uploadConcurrency
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Error(err)
		}
		err = p.AuthCaptchaToken("POST:/drive/v1/files")
		if err != nil {
			logrus.Error(err)
		}

		go func() {
			ticker := time.NewTicker(time.Second * 7200 * 3 / 4)
			defer ticker.Stop()
			for range ticker.C {
				err := p.RefreshToken()
				if err != nil {
					logrus.Warn(err)
					continue
				}
			}
		}()
		for _, v := range args {
			stat, err := os.Stat(v)
			if err != nil {
				logrus.Errorf("Get file %s stat failed: %s", v, err)
				continue
			}
			if stat.IsDir() {
				handleUploadFolder(&p, v)
			} else {
				handleUploadFile(&p, v)
			}
		}
	},
}

// Specifies the folder of the pikpak server
var uploadFolder string

// Specifies the file to upload
var uploadConcurrency int64

// Sync mode
var sync bool

// Init upload command
func init() {
	UploadCmd.Flags().StringVarP(&uploadFolder, "path", "p", "/", "specific the folder of the pikpak server")
	UploadCmd.Flags().Int64VarP(&uploadConcurrency, "concurrency", "c", 1<<4, "specific the concurrency of the upload")
	UploadCmd.Flags().StringSliceVarP(&exclude, "exn", "e", []string{}, "specific the exclude file or folder")
	UploadCmd.Flags().BoolVarP(&sync, "sync", "s", false, "sync mode")
}

// Exclude string list
var exclude []string

var defaultExcludeRegexp []*regexp.Regexp = []*regexp.Regexp{
	// exclude the hidden file
	regexp.MustCompile(`^\..+`),
}

// Dispose the exclude file or folder
func disposeExclude() {
	for _, v := range exclude {
		defaultExcludeRegexp = append(defaultExcludeRegexp, regexp.MustCompile(v))
	}
}
func handleUploadFile(p *pikpak.PikPak, path string) {
	parentId, err := p.GetDeepFolderOrCreateId("", uploadFolder)
	if err != nil {
		logrus.Errorf("Get folder %s id failed: %s", uploadFolder, err)
		return
	}
	dir := filepath.Dir(path)

	if dir != "." {
		parentId, err = p.GetDeepFolderOrCreateId(parentId, dir)
		if err != nil {
			logrus.Errorf("Get folder %s id failed: %s\n", dir, err)
			return
		}
	}
	err = p.UploadFile(parentId, path)
	if err != nil {
		logrus.Errorf("Upload file %s failed: %s\n", path, err)
		return
	}
	logrus.Infof("Upload file %s success!\n", path)
}

// upload files logic
func handleUploadFolder(p *pikpak.PikPak, path string) {
	uploadFilePath := utils.GetUploadFilePath(path, defaultExcludeRegexp)

	var f *os.File

	// sync mode
	if sync {
		file, err := os.OpenFile(filepath.Join(".", ".pikpaksync.txt"), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		f = file
		bs, err := ioutil.ReadAll(f)
		if err != nil {
			logrus.Error("read file error: ", err)
			os.Exit(1)
		}
		alreadySyncFiles := strings.Split(string(bs), "\n")
		files := make([]string, 0)
		for _, f := range uploadFilePath {
			if !utils.Contains(alreadySyncFiles, f) {
				files = append(files, f)
			}
		}
		uploadFilePath = files
	}

	logrus.Info("upload file list:")
	for _, f := range uploadFilePath {
		logrus.Infoln(f)
	}

	// if uploadFolder != "" {
	// 	parentPathS := strings.Split(uploadFolder, "/")
	// 	for i, v := range parentPathS {
	// 		if v == "." {
	// 			parentPathS = append(parentPathS[:i], parentPathS[i+1:]...)
	// 		}
	// 	}
	// 	id, err := p.GetDeepFolderOrCreateId(parentId, parentPathS)
	// 	if err != nil {
	// 		logrus.Error(err)
	// 		os.Exit(-1)
	// 	} else {
	// 		parentId = id
	// 	}
	// }
	parentId, err := p.GetDeepFolderOrCreateId("", uploadFolder)
	if err != nil {
		logrus.Errorf("get folder %s id error: ", uploadFolder, err)
	}

	logrus.Debug("upload folder: ", uploadFolder, " parentId: ", parentId)

	for _, v := range uploadFilePath {
		if strings.Contains(v, "/") || strings.Contains(v, "\\") {
			basePath := filepath.Dir(v)
			id, err := p.GetDeepFolderOrCreateId(parentId, basePath)
			if err != nil {
				logrus.Error(err)
			}
			err = p.UploadFile(id, filepath.Join(path, v))
			if err != nil {
				logrus.Error(err)
			}
			if sync {
				f.WriteString(v + "\n")
			}
			logrus.Infof("%s upload success!\n", v)
		} else {
			err = p.UploadFile(parentId, filepath.Join(path, v))
			if err != nil {
				logrus.Error(err)
			}
			if sync {
				f.WriteString(v + "\n")
			}
		}
	}
}
