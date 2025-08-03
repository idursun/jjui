package test

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"reflect"
	"time"
)

type teaModelLike interface {
	Init() tea.Cmd
	View() string
}

// model is a generic shell that can wrap any teaModelLike
type model[T teaModelLike] struct {
	closed        bool
	embeddedModel T
}

func (m model[T]) Init() tea.Cmd {
	return m.embeddedModel.Init()
}

func (m model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case common.CloseViewMsg, confirmation.CloseMsg:
		m.closed = true
		// give enough time to clear pending messages before quitting
		return m, tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
			return tea.QuitMsg{}
		})
	default:
		updatedModel, cmd := callUpdate(m.embeddedModel, msg)
		if updatedModel != nil {
			m.embeddedModel = updatedModel.(T)
		}
		return m, cmd
	}
}

// callUpdate this is acting as an adapter to call the Update method on the embedded model
// embeddedModels don't have to implement the Update method as `Update(msg tea.Msg) (tea.Model, tea.Cmd)`
func callUpdate(model interface{}, msg tea.Msg) (interface{}, tea.Cmd) {
	modelValue := reflect.ValueOf(model)

	// Find the Update method
	updateMethod := modelValue.MethodByName("Update")
	if !updateMethod.IsValid() {
		// No Update method found
		return model, nil
	}

	results := updateMethod.Call([]reflect.Value{reflect.ValueOf(msg)})
	if len(results) != 2 {
		return model, nil
	}

	updatedModel := results[0].Interface()

	var cmd tea.Cmd
	if !results[1].IsNil() {
		cmd = results[1].Interface().(tea.Cmd)
	}

	return updatedModel, cmd
}

func (m model[T]) View() string {
	if m.closed {
		return "closed"
	}
	return m.embeddedModel.View()
}

// NewShell creates a new shell wrapping the provided model
func NewShell[T teaModelLike](embeddedModel T) tea.Model {
	return model[T]{
		embeddedModel: embeddedModel,
	}
}
