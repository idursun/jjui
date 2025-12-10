package askpass

func (s *Server) DebugSubprocessCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.subprocesses)
}
