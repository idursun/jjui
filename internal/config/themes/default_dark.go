package themes

import "github.com/idursun/jjui/internal/config"

var DarkTheme = map[string]config.Color{
	"dimmed":                     {Fg: "bright black"},
	"title":                      {Fg: "magenta", Bold: true},
	"text":                       {Fg: "white"},
	"shortcut":                   {Fg: "magenta"},
	"matched":                    {Fg: "cyan"},
	"selected":                   {Fg: "cyan", Bg: "bright black"},
	"target_marker":              {Fg: "black", Bg: "red", Bold: true},
	"source_marker":              {Fg: "black", Bg: "cyan"},
	"success":                    {Fg: "green"},
	"error":                      {Fg: "red"},
	"border":                     {Fg: "bright white"},
	"confirmation text":          {Fg: "magenta", Bold: true},
	"confirmation selected":      {Fg: "bright white", Bg: "blue", Bold: true},
	"confirmation dimmed":        {Fg: "white"},
	"help title":                 {Fg: "green", Bold: true},
	"details selected":           {Bg: "bright black"},
	"revset title":               {Fg: "magenta"},
	"revset text":                {Fg: "green", Bold: true},
	"revset completion text":     {Fg: "white"},
	"revset completion matched":  {Fg: "cyan", Bold: true},
	"revset completion selected": {Fg: "cyan", Bg: "bright black"},
	"status title":               {Fg: "black", Bg: "magenta", Bold: true},
	"menu title":                 {Fg: "230", Bg: "62", Bold: true},
	"menu matched":               {Fg: "magenta", Bold: true},
	"menu selected":              {Fg: "cyan", Bold: true},
}
