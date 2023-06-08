package filter

type Filter struct {
}

func NewFilter() *Filter {
	return &Filter{}
}

// AddRemoteFilters parses the hosts files at the given URLs and adds the hosts
// listed in them to the filter.
func (f *Filter) AddRemoteFilters(urls []string) error {
	return nil
}

// AddRemoteFilter parses the hosts file at the given URL and adds the hosts
// listed in it to the filter.
func (f *Filter) AddRemoteFilter(url string) error {
	return nil
}

func (f *Filter) addHost(host string) {
}

// IsBlocked returns true if the given host is blocked by the filter.
func (f *Filter) IsBlocked(host string) bool {
	return false
}
