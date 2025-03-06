import org.jenkinsci.plugins.pipeline.github.trigger.IssueCommentCause

@Library('pipeline-lib') _
@Library('devsec-lib') __

def MASTER_BRANCH = 'master'
def PROJECT_NAME = 'github-review-helper'

properties([
  pipelineTriggers([issueCommentTrigger('!build')])
])

withResultReporting(slackChannel: '#tm-devexp') {
  withImageBuilder(
    containers: [
      interactiveContainer(name: 'go', image: '662491802882.dkr.ecr.us-east-1.amazonaws.com/golang:1.23-20250306-b193'),
      devsec.imageScannerContainer()
    ]
  ) {
    checkout(scm)

    def isMaster = env.BRANCH_NAME == MASTER_BRANCH
    def isPR = !!env.CHANGE_ID
    def pushImages = isMaster || !!currentBuild.rawBuild.getCause(IssueCommentCause)

    def image

    stage('Build image') {
      image = imageBuilder.build(PROJECT_NAME)
    }

    stage('Scan docker image') {
      devsec.scanImageFile(image.imageFileName())
    }

    def imageVersion
    if (env.BRANCH_NAME == MASTER_BRANCH) {
      imageVersion = sh(returnStdout: true, script: 'date +%Y%m%d').trim()
    } else {
      imageVersion = sh(returnStdout: true, script: 'git rev-parse --short HEAD').trim()
      imageVersion = "pr-${imageVersion}"
    }

    stage('Publish docker image') {
      imageBuilder.withECR {
        if (pushImages) {
          echo("Publishing docker image ${image.imageName()} with tag ${imageVersion} and latest")
          image.push("${imageVersion}")

          if (isMaster) {
            image.push("latest")
          }

          if (isPR && pushImages) {
            pullRequest.comment("Built and published `${imageVersion}`")
          }
        } else {
          echo("${env.BRANCH_NAME} is not the master branch. Not publishing the docker image.")
        }
      }
    }
  }
}
