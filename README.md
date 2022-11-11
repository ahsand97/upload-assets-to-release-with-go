# ahsand97/upload-assets-to-release-with-go
Cross platform GitHub action to upload multiple assets to a release using Golanguage.

## Features
- Assets can be global regex
- Overwrite assets if wanted
- Reverse (delete all uploaded assets) in case of failure if wanted
- Output with colors and emojis

## Requirements
This action could fail if there's no write permissions on contents.

```yml
permissions:
  contents: write
```

## Arguments
All default values are passed if the action is used on a `release` event workflow, otherwise the missing ones need to be provided since the `event` associated might not have all the necessary values.

|Name|Required|Default value|Description|
|:---:|:---:|:---:|:---|
|`files`|**always**||Paths of the assets to be uploaded, it can be glob regex. **It must be a string array**. For example: `files: '["my_asset", "*.py", "dist/*"]'`|
|`token`|no|`github.token`|GitHub Acess Token. Picked automatically from `github` context.|
|`tag`|yes|`github.event.release.tag_name`|Tag associated with the release where to upload the assets. Picked automatically from `github` context if the `event` that triggered the workflow is `release`, if not, **it must be provided**.|
|`owner`|no|`github.repository_owner`|Owner of Respository. Picked automatically from `github` context.|
|`repo`|no|`github.repository`|Repository where to upload assets. Picked automatically from `github` context.|
|`workspace`|no|`github.workspace`|Workspace where to search the assets. Picked utomatically from `github` context.|
|`overwrite_assets`|no|`true`|Overwrite assets if they're already in the release.|
|`revert_on_failure`|no|`true`|Revert (delete all already uploaded assets) in case of failure.|

## Example
```yml
name: Test Upload Assets To Release on Release Event

on:
  release:
    types: [published]

jobs:
  test-upload-assets:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v3
      - name: Upload Assets to Release with Go
        uses: ahsand97/upload-assets-to-release-with-go@v0.1.0
        with:
          files: '["my_asset"]'
```
## License
This project is released under the [MIT](https://github.com/ahsand97/upload-assets-to-release-with-go/blob/main/LICENSE) license.