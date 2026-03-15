module audio

go 1.25.0

require (
	github.com/dhowden/tag v0.0.0-20240417053706-3d75831295e8
	github.com/spf13/pflag v1.0.10
	metadata v0.0.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace metadata => ../..
