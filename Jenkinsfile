#!groovy

node {
    stage 'Test stuff'
    stage 'Build static binary' 
    sh '''
        docker run --rm -v $PWD:/go/src/github.com/UnityTech/kafkalogsforwarder -w /go/src/github.com/UnityTech/kafkalogsforwarder golang:latest /bin/bash -c "go get -u github.com/kardianos/govendor; govendor sync; go build -a -ldflags \'-s\' -tags netgo -installsuffix netgo -v -o kafkalogsforwarder && ! ldd kafkalogsforwarder"
    '''

    if (env.BRANCH_NAME != "master") {
        stage 'Print stuff'
        echo "No need to do anything."
    }

    if (env.BRANCH_NAME == "master") {
        stage 'Build the image'
        withEnv([
            "service=${env.JOB_NAME.split('/')[0]}",
            "branch=${env.BRANCH_NAME}",
            "revision=${sh([returnStdout: true, script: 'git log --format=\"%H\" -n 1']).trim()}",
            'docker_image="registry2.applifier.info:5005/$service:$revision',
        ]) {
            sh '''
                env
            '''
        }
    }
}
