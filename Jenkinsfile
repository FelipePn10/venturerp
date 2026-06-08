pipeline {
    // Per-stage agents so the Go toolchain and Qodana each run in their own image.
    agent none

    options {
        timestamps()
        timeout(time: 30, unit: 'MINUTES')
    }

    stages {
        stage('Build & Test') {
            agent {
                docker {
                    image 'golang:1.25'
                    args '-v go-mod-cache:/go/pkg/mod'
                }
            }
            environment {
                // Vendored build; keep Go's cache inside the workspace so the agent can write it.
                GOFLAGS = '-mod=vendor'
                GOCACHE = "${WORKSPACE}/.gocache"
            }
            steps {
                sh 'go version'
                sh 'make fmt-check'
                sh 'make vet'
                sh 'make build'
                sh 'make test-cover'
            }
        }

        stage('Qodana') {
            when {
                anyOf {
                    branch 'main'
                    branch 'wip/create_product_structure'
                }
            }
            agent {
                docker {
                    image 'jetbrains/qodana-go'
                    args '''
                        -v "${WORKSPACE}":/data/project
                        --entrypoint=""
                        '''
                }
            }
            environment {
                QODANA_TOKEN = credentials('qodana-token')
            }
            steps {
                sh '''qodana'''
            }
        }
    }
}
