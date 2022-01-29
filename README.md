<h1 align="center">Yagma</h1>
<h3 align="center">Yet Another Go Mojang API</h3>
<p align="center">
  <a href="https://github.com/alteamc/yagma/actions/workflows/go.yml"><img alt="Workflow status" src="https://img.shields.io/github/workflow/status/alteamc/yagma/Go/master"></a>
  <a href="https://app.codacy.com/gh/alteamc/yagma"><img alt="Codacy grade" src="https://img.shields.io/codacy/grade/8c50344d066645af948bfbc0a9b51017"></a>
  <a href="https://github.com/alteamc/yagma/blob/master/go.mod"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/alteamc/yagma"></a>
  <a href="https://github.com/alteamc/yagmma/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/alteamc/yagma"></a>
  <a href="https://pkg.go.dev/github.com/alteamc/yagma"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/alteamc/yagma.svg"></a>
  <a href="https://github.com/alteamc/yagma/blob/master/LICENSE"><img alt="License" src="https://img.shields.io/github/license/alteamc/yagma"></a>
  <a href="https://discord.gg/9ruheUG3Wg"><img alt="License" src="https://img.shields.io/discord/929337829610369095"></a>
</p>

# Mojang API support

While we plan on wrapping Mojang API entirely, Yagma remains mostly an internal tool we use in our other projects and
thus certain endpoints will have higher priority for us to implement.

Here's a support table for your convenience.

| Endpoint                          | Support status                                                                                                                                                                                        |
|-----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Username to UUID                  | Supported ✓                                                                                                                                                                                           |
| Usernames to UUID                 | Supported ✓                                                                                                                                                                                           |
| UUID to Name History              | Unsupported ✗                                                                                                                                                                                         |
| UUID to Profile and Skin/Cape     | Supported ✓                                                                                                                                                                                           |
| Blocked Servers                   | Unsupported ✗                                                                                                                                                                                         |
| Statistics                        | Unsupported ✗                                                                                                                                                                                         |
| Profile information               | Unsupported ✗                                                                                                                                                                                         |
| Player Attributes                 | Unsupported ✗                                                                                                                                                                                         |
| Profile Name Change Information   | Unsupported ✗                                                                                                                                                                                         |
| Check Product Voucher             | Unsupported ✗                                                                                                                                                                                         |
| Name Availability                 | Unsupported ✗                                                                                                                                                                                         |
| Change Name                       | Unsupported ✗                                                                                                                                                                                         |
| Change Skin                       | Unsupported ✗                                                                                                                                                                                         |
| Upload Skin                       | Unsupported ✗                                                                                                                                                                                         |
| Reset Skin                        | Unsupported ✗                                                                                                                                                                                         |
| Hide Cape                         | Unsupported ✗                                                                                                                                                                                         |
| Show Cape                         | Unsupported ✗                                                                                                                                                                                         |
| Show Cape                         | Unsupported ✗                                                                                                                                                                                         |
| Verify Security Location          | Unsupported ✗                                                                                                                                                                                         |
| Get Security Questions            | Unsupported ✗                                                                                                                                                                                         |
| Send Security Answers             | Unsupported ✗                                                                                                                                                                                         |
| Get Account Migration Information | Unsupported ✗                                                                                                                                                                                         |
| Account Migration OTP             | Unsupported ✗                                                                                                                                                                                         |
| Verify Account Migration OTP      | Unsupported ✗                                                                                                                                                                                         |
| Submit Migration Token            | Unsupported ✗                                                                                                                                                                                         |
| Connect Xbox Live                 | Unsupported ✗                                                                                                                                                                                         |
