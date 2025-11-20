package bookmark

//func TestSetBookmarkModel_Update(t *testing.T) {
//	commandRunner := test.NewTestCommandRunner(t)
//	commandRunner.Expect(jj.BookmarkListMovable("revision"))
//	commandRunner.Expect(jj.BookmarkSet("revision", "name"))
//	defer commandRunner.Verify()
//
//	op := NewSetBookmarkOperation(test.NewTestContext(commandRunner), "revision")
//	tm := teatest.NewTestModel(t, op)
//	tm.Type("name")
//	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return commandRunner.IsVerified()
//	})
//	tm.Quit()
//	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
//}
