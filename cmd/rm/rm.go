package rm

import (
	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var long bool
var human bool
var path string
var parentId string

var RmCmd = &cobra.Command{
	Use:   "rm",
	Short: `Remove folder or file`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
		}
		handle(&p, args)
	},
}

func init() {
	RmCmd.Flags().StringVarP(&path, "path", "p", "/", "display the specified path")
	RmCmd.Flags().StringVarP(&parentId, "parent-id", "P", "", "display the specified parent id")
}

func handle(p *pikpak.PikPak, args []string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logrus.Errorln("get path folder id error:", err)
			return
		}
	}
	err = p.RemoveFolder(parentId)
	if err != nil {
		logrus.Errorln("remove folder error:", err)
		return
	}
	err = p.TrashEmpty()
	if err != nil {
		logrus.Errorln("remove folder error:", err)
		return
	}
}

