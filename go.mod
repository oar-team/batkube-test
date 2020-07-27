module github.com/oar-team/batkube-test

go 1.14

require (
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/mitchellh/mapstructure v1.3.2
	github.com/sirupsen/logrus v1.6.0
	gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube v0.0.0-00010101000000-000000000000
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/utils v0.0.0-20200720150651-0bdb4ca86cbc // indirect
)

replace (
	gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube => gitlab.com/ryax-tech/internships/2020/scheduling_simulation/batkube.git v0.0.0-20200724163848-09dd32cf5c0d
	k8s.io/api => k8s.io/api v0.18.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.2-beta.0
	k8s.io/client-go => k8s.io/client-go v0.18.0
)
