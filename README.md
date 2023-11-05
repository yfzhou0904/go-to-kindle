# go-to-kindle
parse webpage into article and send to kindle

# Requirements
go v1.21 or newer

# Install & configuration
```sh
make
cp example_config.toml ~/.go-to-kindle/config.toml
vim ~/.go-to-kindle/config.toml # edit email credential
```

# Usage
```sh
./bin/go-to-kindle <web article url>
```