package reactivetools

func NewCheckResult(objectType, objectID, checkTypeName, result string, success bool) CheckResult {
	return &checkResult{
		ot:  objectType,
		oid: objectID,
		cn:  checkTypeName,
		rm:  result,
		cs:  success,
	}
}

type checkResult struct {
	ot  string
	oid string
	cn  string
	rm  string
	cs  bool
}

func (c *checkResult) ObjectType() string {
	return c.ot
}

func (c *checkResult) ObjectIdentifier() string {
	return c.oid
}

func (c *checkResult) CheckName() string {
	return c.cn
}

func (c *checkResult) ResultMessage() string {
	return c.rm
}

func (c *checkResult) CheckSuccess() bool {
	return c.cs
}
