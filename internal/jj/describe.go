package jj

type DescribeArgs struct {
	Revisions   SelectedRevisions
	Description string
}

func (d DescribeArgs) GetArgs() CommandArgs {
	args := []string{"describe", "--edit"}
	args = append(args, d.Revisions.AsArgs()...)
	return args
}

func Describe(revisions SelectedRevisions) CommandArgs {
	args := []string{"describe", "--edit"}
	args = append(args, revisions.AsArgs()...)
	return args
}

func SetDescription(revision string, description string) CommandArgs {
	return []string{"describe", "-r", revision, "-m", description}
}

func GetDescription(revision string) CommandArgs {
	return []string{"log", "-r", revision, "--template", "description", "--no-graph", "--ignore-working-copy", "--color", "never", "--quiet"}
}
