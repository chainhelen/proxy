#include <stdio.h>
#include <unistd.h>
#include <fcntl.h> 
#include <string.h>
#include <sys/epoll.h> 
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>
#include <stdlib.h>

const int timeout = -1;
const int maxNumEvt = 64;
const int maxNumFd = 256;
const int cpuNum = 20;
const int port = 80;
const char *host = "127.0.0.1";

void set_nonblock(int fd)  
{  
    int fl = fcntl(fd,F_GETFL);  
    fcntl(fd,F_SETFL,fl | O_NONBLOCK);  
}  

int start() {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) {
        perror("sock");
        exit(2);
    }

    int opt = 1;
    setsockopt(sock,SOL_SOCKET,SO_REUSEADDR,&opt,sizeof(opt));

    struct sockaddr_in local;         
    local.sin_port=htons(port);
    local.sin_family=AF_INET;  
    local.sin_addr.s_addr=inet_addr(host); 

    if(bind(sock, (struct sockaddr*)&local, sizeof(local)) < 0) 
    {
        perror("bind");
        exit(3);
    }

    if(listen(sock, 5) < 0) 
    {
        perror("listen");
        exit(4);
    }
    return sock;
}

int startWorkProcess(int listen_fd, pid_t pid) 
{
    (void)listen_fd;
    (void)pid;

    //1.创建epoll      
    int epfd = epoll_create(maxNumFd);    //可处理的最大句柄数256个  
    if(epfd < 0)  
    {  
        perror("epoll_create");  
        exit(5);  
    }  

    struct epoll_event _ev;       //epoll结构填充   
    _ev.events = EPOLLIN;         //初始关心事件为读  
    _ev.data.fd = listen_fd;     

    //2.托管  
    epoll_ctl(epfd,EPOLL_CTL_ADD,listen_fd,&_ev);  //将listen sock添加到epfd中，关心读事件  

    struct epoll_event revs[maxNumEvt];


    for(;;)  
    {  
        //epoll_wait()相当于在检测事件  
        int num = epoll_wait(epfd, revs, maxNumEvt, timeout);
        if (0 == num) {
            printf("timeout \n");
            continue;
        }
        if (-1 == num) 
        {
            perror("epoll_wait");
            continue;
        }
        // Get Event
        struct sockaddr_in peer;  
        socklen_t len = sizeof(peer);  

        for(int i=0;i < num;i++)  
        {  
            int rsock = revs[i].data.fd; //准确获取哪个事件的描述符  
            if(rsock == listen_fd && (revs[i].events) && EPOLLIN) //如果是初始的 多个worker就竞争接受，建立链接 
            {  
                int sock_fd = accept(listen_fd,(struct sockaddr*)&peer,&len);    

                if(sock_fd > 0)  
                {  
                    printf("get a new client:%s:%d\n",inet_ntoa(peer.sin_addr),ntohs(peer.sin_port));  

                    set_nonblock(sock_fd);  
                    _ev.events = EPOLLIN | EPOLLET;  
                    _ev.data.fd = sock_fd; 
                    epoll_ctl(epfd,EPOLL_CTL_ADD,sock_fd,&_ev);    //二次托管  
                }  
            }  
            else // 接下来对num - 1 个事件处理  
            {  
                if(revs[i].events & EPOLLIN)  
                {  
                    char buf[1024];  
                    ssize_t _s = read(rsock,buf,sizeof(buf)-1);  
                    if(_s > 0)  
                    {  
                        buf[_s] = '\0';  
                        printf("client:%s\n",buf);  

                        _ev.events = EPOLLOUT | EPOLLET;  
                        _ev.data.fd = rsock;  
                        epoll_ctl(epfd,EPOLL_CTL_DEL,rsock,&_ev);    //二次托管  
                    }  
                    else if(_s == 0)  //client:close  
                    {  
                        printf("client:%d close\n",rsock);  
                        epoll_ctl(epfd,EPOLL_CTL_DEL,rsock,NULL);  

                        close(rsock);  
                    }  
                    else  
                    {  
                        perror("read");  
                    }  
                }  
            }  
        }  
    }  
    return 0;
}

int main(int argc,char *argv[])                           
{  
    (void)argc;
    (void)argv;


    int listen_sock=start();      //创建一个绑定了本地 ip 和端口号的套接字描述符  

    pid_t pid;
    for (int i = 0;i < cpuNum;i++) {
        // 创建子进程  
        pid = fork();
        // 这里有子进程创建子子进程的坑，需要break
        if (0 == pid || -1 == pid) {
            break;
        }

    }
    //TO 僵尸进程
    if (-1 == pid) {
        perror("fork failed");
        exit(6);
    }
    if (0 == pid) { // worker
        startWorkProcess(listen_sock, getpid());
    } else { // master
    }

    return 0;  
} 
