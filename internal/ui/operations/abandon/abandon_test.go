package abandon

//var commit = &jj.Commit{ChangeId: "a"}
//var revisions = jj.NewSelectedRevisions(commit)
//
//func Test_Accept(t *testing.T) {
//	commandRunner := test.NewTestCommandRunner(t)
//	commandRunner.Expect(jj.Abandon(revisions, false))
//	defer commandRunner.Verify()
//
//	model := NewOperation(test.NewTestContext(commandRunner), revisions)
//	model.SetSelectedRevision(commit)
//
//	tm := teatest.NewTestModel(t, model)
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return bytes.Contains(bts, []byte("abandon"))
//	})
//
//	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return commandRunner.IsVerified()
//	})
//	tm.Quit()
//	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
//}
//
//func Test_Cancel(t *testing.T) {
//	commandRunner := test.NewTestCommandRunner(t)
//	defer commandRunner.Verify()
//
//	model := NewOperation(test.NewTestContext(commandRunner), revisions)
//	model.SetSelectedRevision(commit)
//
//	tm := teatest.NewTestModel(t, model)
//	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return commandRunner.IsVerified()
//	})
//	tm.Quit()
//	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
//}
