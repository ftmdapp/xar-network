# xar collect-gentxs


## Description
Collect genesis txs and output a genesis.json file

## Usage
```shell
xar collect-gentxs [flags]
```

## Flags
| Name，shorthand| Type  | Default                   | description                   | Required |
| ----------- | ------ | ------------------------- | ------------------------------ | -------- |
| --gentx-dir | string | ~/.xar/config/gentx/ |  override default "gentx" directory from which collect and execute genesis transactions| false  |
| -h, --help  |        |                           |  help for collect-gentxs                    | false  |
| --home      | string | ~/.xar               |  directory for config and data              | false  |
| --trace     | bool   |                           | print out full stack trace on erro          | false  |

## Example
`xar collect-gentxs`
