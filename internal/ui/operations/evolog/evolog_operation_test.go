package evolog

//var revision = &jj.Commit{
//	ChangeId:      "abc",
//	IsWorkingCopy: false,
//	Hidden:        false,
//	CommitId:      "123",
//}

//func TestNewOperation_Mode(t *testing.T) {
//	tests := []struct {
//		name      string
//		mode      mode
//		isFocused bool
//		isOverlay bool
//	}{
//		{
//			name:      "select mode is editing",
//			mode:      selectMode,
//			isFocused: true,
//			isOverlay: true,
//		},
//		{
//			name:      "restore mode is not editing",
//			mode:      restoreMode,
//			isFocused: true,
//			isOverlay: false,
//		},
//	}
//	for _, args := range tests {
//		t.Run(args.name, func(t *testing.T) {
//			commandRunner := test.NewTestCommandRunner(t)
//			context := test.NewTestContext(commandRunner)
//			operation := NewOperation(context, revision, 10, 20)
//			operation.mode = args.mode
//
//			assert.Equal(t, args.isFocused, operation.IsFocused())
//			assert.Equal(t, args.isOverlay, operation.IsOverlay())
//		})
//	}
//}

//func TestOperation_Init(t *testing.T) {
//	commandRunner := test.NewTestCommandRunner(t)
//	commandRunner.Expect(jj.Evolog(revision.ChangeId))
//	defer commandRunner.Verify()
//
//	context := test.NewTestContext(commandRunner)
//	operation := NewOperation(context, revision, 10, 20)
//	tm := teatest.NewTestModel(t, operation)
//
//	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
//		return commandRunner.IsVerified()
//	})
//	tm.Quit()
//	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
//}
