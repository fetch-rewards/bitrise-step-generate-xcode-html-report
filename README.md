# Generate Xcode test report html

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-generate-xcode-html-report?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-generate-xcode-html-report/releases)

The Step converts xcresult summaries to html reports.

<details>
<summary>Description</summary>

This step will generate html report summaries from your xcresult files. It will also include all of the attachments from your tests.

The step works seamlessly with the official Xcode testing steps. If you use those then you do not need to configure this step in any way because it will automatically discover all of the generated xcresult files.

If you use Fastlane or have script step for your building process then you need to tell this step where to find your xcresult files.
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `xcresult_patterns` | A newline (`\n`) separated list of all of the xcresult files  You do not need to specify the xcresult if your are using the official Xcode test steps. This is only needed if you use Fastlane or script based setup.  This input supports glob patterns. This means you can use exact paths or wildcards. Here are a few examples: ``` /path/to/MyApp.xcresult /path/to/output/folder/*.xcresult /path/to/parent/folder/**/*.xcresult ```  The only requirements are that every pattern must only find xcresult files and they have to be absolute paths. |  |  |
| `test_result_dir` | This is directory where the official Xcode testing steps save their output | required | `$BITRISE_TEST_DEPLOY_DIR` |
| `verbose` | Enable logging additional information for debugging. | required | `false` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BITRISE_HTML_REPORT_DIR` | This folder contains the generated html test reports and their assets. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-generate-xcode-html-report/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-generate-xcode-html-report/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Note: this step's end-to-end tests (defined in e2e/bitrise.yml) are working with secrets which are intentionally not stored in this repo. External contributors won't be able to run those tests. Don't worry, if you open a PR with your contribution, we will help with running tests and make sure that they pass.

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
