# go-to-kindle
parse webpage into article and send to kindle

# Requirements
go v1.21 or newer

# Install & configuration
```sh
go install github.com/yfzhou0904/go-to-kindle@latest
```
or
```sh
make
cp example_config.toml ~/.go-to-kindle/config.toml
vim ~/.go-to-kindle/config.toml # include email credentials
```

# Usage
```sh
go-to-kindle <url>
```
