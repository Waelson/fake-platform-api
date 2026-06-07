package ids

import "fmt"

// Agent generates an agent ID in the format agent-{env}-{role}-{n:03d}.
func Agent(environment, role string, n int) string {
	return fmt.Sprintf("agent-%s-%s-%03d", environment, role, n)
}

func Command(n int) string {
	return fmt.Sprintf("cmd-%03d", n)
}

func Deployment(n int) string {
	return fmt.Sprintf("dep-%03d", n)
}

func Route(n int) string {
	return fmt.Sprintf("route-%03d", n)
}
