package intents

import "github.com/idursun/jjui/internal/jj"

//jjui:bind scope=revisions action=open_annotation
type OpenAnnotation struct {
	Selected *jj.Commit
}

func (OpenAnnotation) isIntent() {}

type AnnotationShow struct {
	ChangeID string
}

func (AnnotationShow) isIntent() {}

//jjui:bind scope=annotation action=move_up set=Delta:-1
//jjui:bind scope=annotation action=move_down set=Delta:1
//jjui:bind scope=annotation action=select_up set=Delta:-1,Select:true
//jjui:bind scope=annotation action=select_down set=Delta:1,Select:true
//jjui:bind scope=annotation action=page_up set=Delta:-1,Page:true
//jjui:bind scope=annotation action=page_down set=Delta:1,Page:true
//jjui:bind scope=annotation action=half_page_up set=Delta:-1,HalfPage:true
//jjui:bind scope=annotation action=half_page_down set=Delta:1,HalfPage:true
type AnnotationMove struct {
	Delta    int
	Select   bool
	Page     bool
	HalfPage bool
}

func (AnnotationMove) isIntent() {}

//jjui:bind scope=annotation action=move_top set=Last:false
//jjui:bind scope=annotation action=move_bottom set=Last:true
type AnnotationMoveBoundary struct {
	Last bool
}

func (AnnotationMoveBoundary) isIntent() {}

//jjui:bind scope=annotation action=prev_file set=Delta:-1
//jjui:bind scope=annotation action=next_file set=Delta:1
type AnnotationFileNavigate struct {
	Delta int
}

func (AnnotationFileNavigate) isIntent() {}

//jjui:bind scope=annotation action=target_picker
type AnnotationOpenTargetPicker struct{}

func (AnnotationOpenTargetPicker) isIntent() {}

//jjui:bind scope=annotation action=comment_picker
type AnnotationOpenCommentPicker struct{}

func (AnnotationOpenCommentPicker) isIntent() {}

//jjui:bind scope=annotation action=parent_revision
type AnnotationNavigateParent struct{}

func (AnnotationNavigateParent) isIntent() {}

//jjui:bind scope=annotation action=child_revision
type AnnotationNavigateChild struct{}

func (AnnotationNavigateChild) isIntent() {}

//jjui:bind scope=annotation action=left set=Delta:-1
//jjui:bind scope=annotation action=right set=Delta:1
type AnnotationScrollHorizontal struct {
	Delta int
}

func (AnnotationScrollHorizontal) isIntent() {}

//jjui:bind scope=annotation action=toggle_presentation
type AnnotationTogglePresentation struct{}

func (AnnotationTogglePresentation) isIntent() {}

//jjui:bind scope=annotation action=toggle_wrap
type AnnotationToggleWrap struct{}

func (AnnotationToggleWrap) isIntent() {}

//jjui:bind scope=annotation action=add
type AnnotationAdd struct{}

func (AnnotationAdd) isIntent() {}

//jjui:bind scope=annotation action=delete
type AnnotationDelete struct{}

func (AnnotationDelete) isIntent() {}

//jjui:bind scope=annotation action=clear
type AnnotationClear struct{}

func (AnnotationClear) isIntent() {}

//jjui:bind scope=annotation action=copy
type AnnotationCopy struct{}

func (AnnotationCopy) isIntent() {}

//jjui:bind scope=annotation.editor action=save
type AnnotationEditorSave struct{}

func (AnnotationEditorSave) isIntent() {}

//jjui:bind scope=annotation.editor action=cancel
type AnnotationEditorCancel struct{}

func (AnnotationEditorCancel) isIntent() {}
