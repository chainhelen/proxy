#include <stdio.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/socket.h>

int main() 
{
    int sockets[2];
    char buf[1024];

    if(socketpair(AF_UNIX, SOCK_STREAM, 0, sockets) < 0) {
        perror("socketpair failed");
        return -1;
    }

    pid_t pid;
    if (-1 == (pid = fork())) {
        perror("fork failed");
        return -1;
    }


    if (0 == pid) {
        printf("I am children\n");
        int n = 0;
        if ((n = read(sockets[1], buf, sizeof(buf))) < 0) {
            perror("child read failed\n");
            return -1;
        }
        printf("[children]");
        for(int i = 0;i < n;i++) {
           printf("%c", buf[i]);
        }
        printf("\n");
    } else {
        printf("I am parent\n");
        if(write(sockets[0], "send message from parent\0", sizeof("send message from parent\0")) < 0) {
            perror("parent write failed");
            return -1;
        }
        const char *pathname = "./shmwrite.c";
        int fd;
        if((fd = open(pathname, O_WRONLY)) < 0) {
            perror("open failed\n");
            return - 1;
        }
    }
    return 0;
}
