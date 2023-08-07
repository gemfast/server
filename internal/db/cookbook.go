package db

import "time"

//	{
//		"name": "yum",
//		"maintainer": "sous-chefs",
//		"description": "Configures various yum components on Red Hat-like systems",
//		"category": "Other",
//		"latest_version": "http://supermarket.chef.io/api/v1/cookbooks/yum/versions/7_1_0",
//		"external_url": "https://github.com/sous-chefs/yum",
//		"average_rating": null,
//		"created_at": "2011-04-20T22:16:12.000Z",
//		"updated_at": "2021-08-29T18:45:03.956Z",
//		"deprecated": false,
//		"versions": [
//			"http://supermarket.chef.io/api/v1/cookbooks/yum/versions/7_1_0",
//			"http://supermarket.chef.io/api/v1/cookbooks/yum/versions/7_0_0"
//		]
//	}
type Cookbook struct {
	Name          string    `json:"name"`
	Maintainer    string    `json:"maintainer"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	LatestVersion string    `json:"latest_version"`
	ExternalURL   string    `json:"external_url"`
	AverageRating any       `json:"average_rating"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Deprecated    bool      `json:"deprecated"`
	Versions      []string  `json:"versions"`
}

//	{
//		"license": "Apache 2.0",
//		"tarball_file_size": 18553,
//		"version": "2.4.0",
//		"average_rating": null,
//		"cookbook": "http://supermarket.chef.io/api/v1/cookbooks/apt",
//		"file": "http://supermarket.chef.io/api/v1/cookbooks/apt/versions/2_4_0/download",
//		"dependencies": {},
//		"platforms": {
//		  "debian": ">= 0.0.0",
//		  "ubuntu": ">= 0.0.0"
//		}
//	  }
type CookbookVersion struct {
	License         string `json:"license"`
	TarballFileSize int    `json:"tarball_file_size"`
	Version         string `json:"version"`
	AverageRating   any    `json:"average_rating"`
	Cookbook        string `json:"cookbook"`
	File            string `json:"file"`
	Dependencies    struct {
	} `json:"dependencies"`
	Platforms struct {
		Debian string `json:"debian"`
		Ubuntu string `json:"ubuntu"`
	} `json:"platforms"`
}
