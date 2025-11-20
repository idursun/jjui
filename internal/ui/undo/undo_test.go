package undo

//func TestConfirm(t *testing.T) {
//	commandRunner := test.NewTestCommandRunner(t)
//	commandRunner.Expect(jj.OpLog(1))
//	commandRunner.Expect(jj.Undo())
//	defer commandRunner.Verify()
//
//	model := NewModel(test.NewTestContext(commandRunner))
//	tm := teatest.NewTestModel(t, model)
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return bytes.Contains(bts, []byte("undo"))
//	})
//	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return commandRunner.IsVerified()
//	})
//	tm.Quit()
//	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
//}
//
//func TestCancel(t *testing.T) {
//	commandRunner := test.NewTestCommandRunner(t)
//	commandRunner.Expect(jj.OpLog(1))
//	defer commandRunner.Verify()
//
//	tm := teatest.NewTestModel(t, NewModel(test.NewTestContext(commandRunner)))
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return bytes.Contains(bts, []byte("undo"))
//	})
//	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return commandRunner.IsVerified()
//	})
//	tm.Quit()
//	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
//}
