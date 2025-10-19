package view

import "github.com/idursun/jjui/internal/jj"

type IStatus interface {
	Name() string
}

type ICommandBuilder interface {
	GetCommand() jj.CommandArgs
}
