pipeline {
    agent any

    environment {
        DOCKERHUB_USER = 'nicolasmartinss'
        IMAGE_NAME     = 'gastano-menos'
        IMAGE_TAG      = "${BUILD_NUMBER}"
        APP_VM_IP      = '201.23.83.151'
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build') {
            steps {
                sh 'docker build -t ${DOCKERHUB_USER}/${IMAGE_NAME}:${IMAGE_TAG} .'
            }
        }

        stage('Push') {
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'dockerhub-credentials',
                    usernameVariable: 'DOCKER_USER',
                    passwordVariable: 'DOCKER_PASS'
                )]) {
                    sh 'echo $DOCKER_PASS | docker login -u $DOCKER_USER --password-stdin'
                    sh 'docker push ${DOCKERHUB_USER}/${IMAGE_NAME}:${IMAGE_TAG}'
                    sh 'docker tag ${DOCKERHUB_USER}/${IMAGE_NAME}:${IMAGE_TAG} ${DOCKERHUB_USER}/${IMAGE_NAME}:latest'
                    sh 'docker push ${DOCKERHUB_USER}/${IMAGE_NAME}:latest'
                }
            }
        }

        stage('Deploy') {
            steps {
                sshagent(['app-vm-ssh']) {
                    sh """
                        ssh -o StrictHostKeyChecking=no ubuntu@${APP_VM_IP} '
                            docker pull ${DOCKERHUB_USER}/${IMAGE_NAME}:latest &&
                            docker stop gastano-menos || true &&
                            docker rm gastano-menos || true &&
                            docker run -d \
                                --name gastano-menos \
                                --restart always \
                                -p 8080:8080 \
                                --env-file /app/gastano-menos.env \
                                ${DOCKERHUB_USER}/${IMAGE_NAME}:latest
                        '
                    """
                }
            }
        }
    }

    post {
        success {
            echo "Deploy realizado com sucesso! Build #${BUILD_NUMBER}"
        }
        failure {
            echo "Pipeline falhou no build #${BUILD_NUMBER}"
        }
    }
}