package blog

// Simple struct to capture io.Write()
// Used to capture output of rendered html/template

type Capture struct {
  rendered string
}

func (c *Capture) Write(p []byte) (n int, err error) {
  c.rendered = c.rendered + string(p)
  return len(p), nil
}

func (c Capture) Render() string {
  return c.rendered
}
