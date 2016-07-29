# SVN GIT Migration Utility

This SVN -> Git migration utility is a clone of the one already available [here](https://www.atlassian.com/git/tutorials/svn-to-git-prepping-your-team-migration)

### Commands:
* verify: This command verifies if the required tools like git & svn are installed and at the latest version

* authors: This command prints the list of authors in the SVN repo to a file
  * -domain: This is the domain of your company.
  The email IDs will be printed according to this domain. _Default_: foo.com
  * -repo: The absolute location of the SVN repo on the local filesystem. _Default_: /home/svn/repo
  * -filename: The filename to which the authors list will be printed. _Default_: foo.com

