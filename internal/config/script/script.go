package script

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type ScriptConfig struct {
	Script map[string]Script `toml:"script"`
}

type Script struct {
	Name     string                   `toml:"name"`
	Steps    []IScriptStep            `toml:"-"`
	RawSteps []map[string]interface{} `toml:"steps"`
}

type IScriptStep interface {
	isScriptStep()
}

var _ IScriptStep = (*JJStep)(nil)
var _ IScriptStep = (*UIStep)(nil)

func (JJStep) isScriptStep() {}
func (UIStep) isScriptStep() {}

type JJStep struct {
	JJ []string `toml:"jj"`
}

type UIStep struct {
	UI struct {
		Action string                 `toml:"action"`
		Params map[string]interface{} `toml:"params,omitempty"`
	} `toml:"ui"`
}

func Parse(data []byte) (*Script, error) {
	config := &ScriptConfig{}
	err := toml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	// For now, just return the first script found
	for name, script := range config.Script {
		// Process raw steps into actual step types
		err = processRawSteps(&script)
		if err != nil {
			return nil, fmt.Errorf("error processing steps for script %s: %v", name, err)
		}
		return &script, nil
	}

	return nil, nil
}

// Process the raw steps into typed steps
func processRawSteps(script *Script) error {
	script.Steps = make([]IScriptStep, 0, len(script.RawSteps))

	for _, rawStep := range script.RawSteps {
		// Check for JJ step
		if jjCmd, ok := rawStep["jj"]; ok {
			// Convert to string slice
			jjCmdSlice, ok := jjCmd.([]interface{})
			if !ok {
				return fmt.Errorf("jj command must be a string array")
			}

			jjStep := &JJStep{
				JJ: make([]string, len(jjCmdSlice)),
			}

			for i, cmd := range jjCmdSlice {
				cmdStr, ok := cmd.(string)
				if !ok {
					return fmt.Errorf("jj command elements must be strings")
				}
				jjStep.JJ[i] = cmdStr
			}

			script.Steps = append(script.Steps, jjStep)
			continue
		}

		// Check for UI step
		if uiData, ok := rawStep["ui"]; ok {
			uiMap, ok := uiData.(map[string]interface{})
			if !ok {
				return fmt.Errorf("ui step must be a map")
			}

			uiStep := &UIStep{}

			// Set action
			action, ok := uiMap["action"]
			if !ok {
				return fmt.Errorf("ui step must have an action")
			}
			actionStr, ok := action.(string)
			if !ok {
				return fmt.Errorf("ui action must be a string")
			}
			uiStep.UI.Action = actionStr

			// Set params if they exist
			if params, ok := uiMap["params"]; ok {
				paramsMap, ok := params.(map[string]interface{})
				if !ok {
					return fmt.Errorf("ui params must be a map")
				}
				uiStep.UI.Params = paramsMap
			} else {
				uiStep.UI.Params = make(map[string]interface{})
			}

			script.Steps = append(script.Steps, uiStep)
			continue
		}

		return fmt.Errorf("unknown step type: %v", rawStep)
	}

	return nil
}
