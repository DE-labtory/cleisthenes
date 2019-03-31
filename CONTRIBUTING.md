### How to contribute

1. Fork [https://github.com/DE-labtory/cleisthenes](https://github.com/DE-labtory/cleisthenes)
2. Register issue you are going to work, and discuss with maintainers whether this feature is needed. Or you can choose existing issue.
3. After assigned the issue, work on it
4. Document about your code
5. Test with `go test -v ./...`
6. Format your code with `goimports -w ./` commands
7. After passing tc and formatting, create pr to `develop` branch referencing issue number
8. Passing travis-ci with more than one approve from maintainers, you can **rebase and merge** to the develop branch
9. After passing all the tc, no build error we can merge to `master` branch at milestone

### Rules to manage branch

* `master`: project with release level can be merged to `master` branch
* `develop`: new feature developed after fully verified from others can be merged to `develop` branch

### Tip

* For not overlapping work with others, please work on small feature as possible.
* When register new issue, concrete documenting issue helps other to understand what you are going to work on and to feedback about your proposal easily.

### Branch naming rule

> [type]/[issue] or [type]/[description]

Sample

> feature/#20

> refac/message

> docs/pbft


### Commit message rule

* Start with lowercase letter.  
* Don't put a `.` at the end of the message.
* Write message authoritatively

#### Format

> [type] : [description]

Sample

> docs : add more description about PBFT 

> fix : change type int to unit


#### Type

- impl : Just implements TODO function or method
- fix : Fix the bug
- chore : Extra work
- feat : A new feature
- perf : A code related performance
- docs : Write or change the docs
- refactor : Only code change 
- test : All about tests (add, fix...)
- critical : A feature related or affect the architecture
