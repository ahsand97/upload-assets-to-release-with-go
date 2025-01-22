package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/chigopher/pathlib"
	"github.com/enescakir/emoji"
	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	context           context.Context
	client            *github.Client
	token             string
	owner             string
	repository        string
	tag               string
	workspace         string
	expectedAssets    []string        // Slice with all valid global regex read from input parameter 'files'
	assetsToUpload    []*pathlib.Path // Slice with all assets that match all global regex from 'expectedAssets' slice
	overwrite_assets  bool            // Whether assets are overwritten or not
	revert_on_failure bool
	release           *github.RepositoryRelease // Release where to upload the assets
}

/*
Get input from environment variables. Necessary env vars:
  - INPUT_OWNER
  - INPUT_TOKEN
  - INPUT_REPO
  - INPUT_TAG
  - INPUT_WORKSPACE
  - INPUT_OVERWRITE_ASSETS
  - INPUT_REVERT_ON_FAILURE
  - INPUT_FILES
*/
func (githubClient *GithubClient) setupDataFromEnv() {
	// Inner function to get data from environment variable
	getDataFromEnv := func(env string, errors *string) string {
		value := os.Getenv(env)
		if len(value) <= 0 {
			*errors += fmt.Sprintf("The environment variable \"%s\" was not found.\n", env)
		}
		return value
	}

	// Errors
	errors := ""

	// Owner
	githubClient.owner = getDataFromEnv("INPUT_OWNER", &errors)

	// Token
	githubClient.token = getDataFromEnv("INPUT_TOKEN", &errors)

	// Repo
	repo := getDataFromEnv("INPUT_REPO", &errors)
	if len(repo) > 0 {
		githubClient.repository = repo
		if strings.Contains(repo, "/") {
			githubClient.repository = strings.Split(repo, "/")[1]
		}
	}

	// Tag
	githubClient.tag = getDataFromEnv("INPUT_TAG", &errors)

	// Workspace
	githubClient.workspace = getDataFromEnv("INPUT_WORKSPACE", &errors)

	// Overwrite Assets
	overwrite_assets := getDataFromEnv("INPUT_OVERWRITE_ASSETS", &errors)
	if len(overwrite_assets) > 0 {
		githubClient.overwrite_assets, _ = strconv.ParseBool(overwrite_assets)
	}

	// Revert on failure
	revert_on_failure := getDataFromEnv("INPUT_REVERT_ON_FAILURE", &errors)
	if len(revert_on_failure) > 0 {
		githubClient.revert_on_failure, _ = strconv.ParseBool(revert_on_failure)
	}

	// Files to upload (glob regex)
	assetsInput := getDataFromEnv("INPUT_FILES", &errors)
	if len(assetsInput) > 0 {
		// Regex to check if input is a valid string array
		regexToCheckInputSyntax := "[(('\\[^'\",\\]+')|(\"\\[^'\",\\]+\")|(\\[^'\",\\]+))(, (('\\[^'\",\\]+')|(\"\\[^'\",\\]+\")|(\\[^'\",\\]+))){0,}]"
		re, _ := regexp.Compile(regexToCheckInputSyntax)
		if !re.MatchString(assetsInput) {
			errors += "There are no assets to upload. The input parameter \"files\" is not a valid string array."
		} else {
			// Regex to extract all valid elements from input
			regExToExtractValidElementsFromInput := "[^,\\[\\]'\"]+"
			re, _ := regexp.Compile(regExToExtractValidElementsFromInput)
			validElementsFromInput := re.FindAllString(assetsInput, -1)
			for _, elementFromInput := range validElementsFromInput {
				elementFromInput = strings.TrimSpace(elementFromInput)
				if len(elementFromInput) <= 0 {
					continue
				}
				elementFromInput = strings.ReplaceAll(elementFromInput, "\"", "")
				elementFromInput = strings.ReplaceAll(elementFromInput, "'", "")
				githubClient.expectedAssets = append(githubClient.expectedAssets, elementFromInput)
			}
		}
	}

	// Show errors
	if len(errors) > 0 {
		fmt.Printf("%v%s", emoji.SadButRelievedFace, fmt.Sprintf(redDarkUnderline, "Error, something went wrong:"))
		fmt.Print("\n")
		errors_ := strings.Split(errors, "\n")
		for _, err_ := range errors_ {
			if len(strings.TrimSpace(err_)) <= 0 {
				continue
			}
			fmt.Printf("%s %s", fmt.Sprintf(yellow, "    -"), fmt.Sprintf(red, err_))
			fmt.Print("\n")
		}
		fmt.Print("\n\n")
		os.Exit(1)
	}

	// If no errors show input info
	fmt.Printf("%s%s", fmt.Sprintf(yellow, "Owner:              "), githubClient.owner)
	fmt.Print("\n")

	fmt.Printf("%s%s", fmt.Sprintf(yellow, "Repo:               "), githubClient.repository)
	fmt.Print("\n")

	fmt.Printf("%s%s", fmt.Sprintf(yellow, "Tag:                "), githubClient.tag)
	fmt.Print("\n")

	fmt.Printf("%s%s", fmt.Sprintf(yellow, "Overwrite assets:   "), strconv.FormatBool(githubClient.overwrite_assets))
	fmt.Print("\n")

	fmt.Printf("%s%s", fmt.Sprintf(yellow, "Revert on failure:  "), strconv.FormatBool(githubClient.revert_on_failure))
	fmt.Print("\n")

	fmt.Printf(yellow, "Assets to upload:")
	fmt.Print("\n")
	for _, expectedAsset := range githubClient.expectedAssets {
		fmt.Printf("%s%s", fmt.Sprintf(yellow, "  - "), expectedAsset)
		fmt.Print("\n")
	}
	fmt.Print("\n")
}

// Setup Github Client
func (githubClient *GithubClient) setupGithubClient() {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubClient.token},
	)
	httpClient := oauth2.NewClient(githubClient.context, tokenSource)
	githubClient.client = github.NewClient(httpClient)
}

// Get related release using the tag defined in env var "INPUT_TAG"
func (githubClient *GithubClient) getReleaseByTag() {
	fmt.Printf(
		blueDark,
		fmt.Sprintf("Searching for related release based on tag \"%s\" ...%v", githubClient.tag, emoji.AlarmClock),
	)
	fmt.Print("\n")
	release, _, err := githubClient.client.Repositories.GetReleaseByTag(githubClient.context,
		githubClient.owner,
		githubClient.repository,
		githubClient.tag,
	)
	if err != nil {
		fmt.Printf(
			"%s %v %s",
			fmt.Sprintf(yellow, "  -"),
			emoji.SadButRelievedFace,
			fmt.Sprintf(redDark, "Error: The release for the tag was not found."),
		)
		fmt.Print("\n\n\n")
		os.Exit(1)
	}
	fmt.Printf("%s Release \"%s\" found %v", fmt.Sprintf(yellow, "  -"), *release.Name, emoji.CheckMark)
	fmt.Print("\n\n")
	githubClient.release = release
}

// Get all matching assets against glob regex input
func (githubClient *GithubClient) getAssetsToUpload() {
	fmt.Printf(blueDark, fmt.Sprintf("Getting path of matching assets...%v", emoji.AlarmClock))
	fmt.Print("\n")

	workspace := pathlib.NewPath(githubClient.workspace)
	var foundAssets []*pathlib.Path // Slice containing all valid assets to upload
	for _, expectedAsset := range githubClient.expectedAssets {
		assetList, _ := workspace.Glob(expectedAsset)
		for _, asset := range assetList {
			if isFile, err := asset.IsFile(); err == nil && isFile {
				foundAssets = append(foundAssets, asset)
			}
		}
	}
	if len(foundAssets) <= 0 {
		fmt.Printf(
			"%s %s",
			fmt.Sprintf(yellow, "  -"),
			fmt.Sprintf(redDark, fmt.Sprintf("There are no matching assets to upload.%v", emoji.ExclamationMark)),
		)
		fmt.Print("\n\n\n")
		os.Exit(1)
	} else {
		for _, foundAsset := range foundAssets {
			fmt.Println(fmt.Sprintf(yellow, "  - ") + foundAsset.String())
		}
		fmt.Print("\n")
		githubClient.assetsToUpload = foundAssets
	}
}

// Upload assets to the release.
// Returns:
//   - bool: Wether all the assets were uploaded to the release or not
func (githubClient *GithubClient) uploadAssetsToRelease() bool {
	resultMain := true
	fmt.Printf(blueDark, fmt.Sprintf("Uploading assets...%v", emoji.AlarmClock))
	fmt.Print("\n")
	const try_number int = 3                              // Times to try to delete/upload an asset
	var successFullyUploadedAssets []*github.ReleaseAsset // Slice containing all assets that are uploaded successfully

	// Auxiliar function to delete an asset from a release
	deleteAsset := func(idAsset int64) {
		for i := 0; i < try_number; i++ {
			res, err := githubClient.client.Repositories.DeleteReleaseAsset(
				githubClient.context,
				githubClient.owner,
				githubClient.repository,
				idAsset,
			)
			if err == nil && res.StatusCode == 204 {
				break
			}
		}
	}

	// Auxiliar function to reverse the uploaded assets in case of failure
	revertAll := func() {
		if len(successFullyUploadedAssets) <= 0 {
			return
		}
		fmt.Print("\n")
		if !githubClient.overwrite_assets {
			fmt.Printf(greenDarkUnderline, "NOTE: Some assets were still uploaded to the release.")
			fmt.Print("\n")
			return
		}
		fmt.Printf(
			"%v  %s  %v",
			emoji.SkullAndCrossbones,
			fmt.Sprintf(
				redDark,
				"REVERTING ALL...DELETING ASSETS THAT WERE UPLOADED TO THE RELEASE",
			),
			emoji.SkullAndCrossbones,
		)
		fmt.Print("\n")
		for _, successfullAsset := range successFullyUploadedAssets {
			deleteAsset(*successfullAsset.ID)
		}
	}

	// Auxiliar function to check if an asset is already uploaded or not in the release by uploading a mock file.
	// If the env var "INPUT_OVERWRITE_FILES" is set and the asset is already in the release the asset gets deleted.
	// Returns:
	//  - bool: Wether the asset can/can't be uploaded
	//  - bool: Wether the asset is already uploaded in the release
	//  - error: Error message
	checkIfAssetCanBeUploaded := func(asset *pathlib.Path) (bool, bool, error) {
		validToUpload := false
		alreadyUploaded := false
		var error error
		mockFile := pathlib.NewPath(os.TempDir()).Join(asset.Name())
		file_, _ := mockFile.Create()
		writer := bufio.NewWriter(file_)
		_, _ = writer.WriteString("MOCK FILE")
		writer.Flush()
		file_.Close()

		// Try to upload mock file
		for i := 0; i < try_number; i++ {
			// Open file
			file, _ := os.Open(mockFile.String())
			releaseAsset, response, err := githubClient.client.Repositories.UploadReleaseAsset(
				githubClient.context,
				githubClient.owner,
				githubClient.repository,
				*githubClient.release.ID,
				&github.UploadOptions{Name: asset.Name()},
				file,
			)
			file.Close()
			error = err
			if response == nil {
				continue
			}
			if response.StatusCode == 201 { // The asset can be uploaded
				deleteAsset(*releaseAsset.ID)
				validToUpload = true
				break
			} else if response.StatusCode == 422 { // The asset is already uploaded to the release
				alreadyUploaded = true
				if githubClient.overwrite_assets { // Try to find asset and delete it to be overwritten
					regExToFindAsset := "[a-zA-Z0-9]*"
					re, _ := regexp.Compile(regExToFindAsset)
					sliceAsset := re.FindAllString(asset.Name(), -1)
					for _, assetInRelease := range githubClient.release.Assets {
						sliceAssetRemote := re.FindAllString(*assetInRelease.Name, -1)
						if reflect.DeepEqual(sliceAsset, sliceAssetRemote) {
							deleteAsset(*assetInRelease.ID)
							validToUpload = true
							alreadyUploaded = false
							break
						}
					}
					if !validToUpload { // Asset not found in the release
						error = fmt.Errorf("NOT FOUND")
					}
				}
				break
			} else if response.StatusCode == 403 { // Forbidden
				error = fmt.Errorf("FORBIDDEN")
				break
			}
		}
		_ = mockFile.Remove()
		return validToUpload, alreadyUploaded, error
	}

	// Auxiliar function to upload a single asset.
	// Returns:
	//  - bool: Wether the file was successfully uploaded or not
	//  - bool: Wether the file was already uploaded to the release or not
	//  - error: Error
	uploadAsset := func(asset *pathlib.Path) (bool, bool, error) {
		result := false

		// Check if asset is valid to be uploaded or not to the release
		isValidToUpload, isAssetAlreadyUploaded, error_ := checkIfAssetCanBeUploaded(asset)

		if !isValidToUpload { // Asset couldn't be uploaded
			if isAssetAlreadyUploaded { // Asset is already uploaded to the release
				result = true
				if error_.Error() == "NOT_FOUND" { // The asset was not found, it couldn't be overwritten
					error_ = fmt.Errorf("the asset could not be overwritten, its id was not found")
					result = false
				}
			} else if error_.Error() == "FORBIDDEN" { // The provided token does not have enough permissions
				error_ = fmt.Errorf("there is not enough permissions to upload the asset")
			}
			return result, isAssetAlreadyUploaded, error_
		}

		// Try to upload the asset
		for i := 0; i < try_number; i++ {
			// Open file
			file, err := os.Open(asset.String())
			if err != nil {
				continue
			}
			// Request to upload asset
			releaseAsset, response, err := githubClient.client.Repositories.UploadReleaseAsset(
				githubClient.context,
				githubClient.owner,
				githubClient.repository,
				*githubClient.release.ID,
				&github.UploadOptions{Name: asset.Name()},
				file,
			)
			file.Close()
			error_ = err
			if response == nil {
				continue
			}
			if response.StatusCode == 201 { // Successfully uploaded
				successFullyUploadedAssets = append(successFullyUploadedAssets, releaseAsset)
				result = true
				break
			}
		}
		return result, isAssetAlreadyUploaded, error_
	}

	// Loop to upload every asset
	for _, assetToUpload := range githubClient.assetsToUpload {
		fmt.Println(fmt.Sprintf(yellow, "  - Asset:  ") + assetToUpload.Name())
		result, isAssetAlreadyUploaded, err := uploadAsset(assetToUpload)
		fmt.Printf(yellow, "    Result: ")
		if result {
			emoji_ := emoji.CheckMark
			text := "The asset has been uploaded successfully."
			if isAssetAlreadyUploaded {
				emoji_ = emoji.CrossMark
				text = "The asset is already in the release."
				resultMain = false
			}
			fmt.Printf("%s %v", text, emoji_)
			fmt.Print("\n\n")
		} else {
			fmt.Printf("%s %v", "The asset could not be uploaded.", emoji.CrossMark)
			fmt.Print("\n\n")
			fmt.Printf("%v %s", emoji.SadButRelievedFace, fmt.Sprintf(redDark, "Error: An error occured uploading the asset."))
			fmt.Print("\n")
			if err != nil {
				fmt.Printf(redDark, err)
				fmt.Print("\n\n")
			}
			revertAll()
			fmt.Print("\n")
			os.Exit(1)
		}
	}
	return resultMain
}

const ( // Colors
	cyanDarkUnderline  = "\033[36;1;4m%s\033[0m"
	blueDark           = "\033[34;1;24m%s\033[0m"
	greenDarkUnderline = "\033[32;1;4m%s\033[0m"
	redDarkUnderline   = "\033[31;1;4m%s\033[0m"
	redDark            = "\033[31;1;24m%s\033[0m"
	yellow             = "\033[33;22;24m%s\033[0m"
	red                = "\033[31;22;24m%s\033[0m"
)

// Main function
func main() {
	fmt.Print("\n\n")
	fmt.Printf(
		"%s %v",
		fmt.Sprintf(cyanDarkUnderline, "UPLOADING ASSETS TO A RELEASE USING GO..."),
		emoji.SmilingFaceWithSunglasses,
	)
	fmt.Print("\n\n")
	fmt.Printf(blueDark, fmt.Sprintf("Hello %v, reading input...%v", emoji.WavingHand, emoji.NerdFace))
	fmt.Print("\n")

	githubClient := &GithubClient{context: context.Background()}
	githubClient.setupDataFromEnv()
	githubClient.setupGithubClient()
	githubClient.getReleaseByTag()
	githubClient.getAssetsToUpload()
	result := githubClient.uploadAssetsToRelease()

	if !result {
		fmt.Printf(redDark, "Note: Some assets were not uploaded to the release because they were already uploaded.")
		fmt.Print("\n\n")
	}

	fmt.Printf(
		"%s%v",
		fmt.Sprintf(
			greenDarkUnderline,
			fmt.Sprintf("The assets have been uploaded to the release \"%s\" successfully.",
				*githubClient.release.Name),
		),
		emoji.NailPolish,
	)
	fmt.Print("\n\n\n")
}
