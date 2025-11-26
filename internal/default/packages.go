package defaults

import service "github.com/flaviogonzalez/instant-layer/internal/services"

var packages = []*service.Package{
	{
		Name: "routes",
	},
	{
		Name: "config",
	},
	{
		Name: "handlers",
	},
	{
		Name: "helpers",
	},
}

func defaultPackages() []*service.Package {

	return packages
}
