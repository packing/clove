/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

/*
#cgo LDFLAGS: -ldl
#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <unistd.h>
#include <errno.h>
#include <sys/socket.h>
#include <sys/stat.h>

typedef struct __daemon_info {
	int         err;
	pid_t       pid;
} daemon_info, *p_daemon_info;

extern void GoPrint(char *s);

void goPrintf(const char *fmt, ...) {
	char szTmpOut[10240];
	va_list vl;
	va_start(vl, fmt);
	vsnprintf(szTmpOut, 10240 - 1, fmt, vl);
	va_end(vl);

	GoPrint(szTmpOut);
}

const char *sfn = NULL;

int beginDaemon(const char *fn)
{
	goPrintf("run pid %d", getpid());
	int fds[2];
	if( socketpair(AF_UNIX,SOCK_STREAM,0,fds) == -1 )
	{
		goPrintf( "socketpair()" );
		exit(-10);
	}

	pid_t pid, sid;
	// Fork off the parent process
	pid = fork();
	if (pid < 0) {
		goPrintf("EXIT_FAILURE 1");
		exit(EXIT_FAILURE);
	}
	// If we got a good PID, then we can exit the parent process.
	if(pid > 0)
	{
		close(fds[1]);
		daemon_info ret;
		ssize_t r = read( fds[0], &ret, sizeof(ret) );
		if( r == sizeof(ret) )
		{
			goPrintf(" >>>> daemon start err:%d  pid:%d\n", ret.err, ret.pid );
			exit(ret.err);
		} else {
			exit(-100);
		}
	}

	close(fds[0]);

	// Change the file mode mask
	umask(0177);
	// Create a new SID for the child process
	sid = setsid();
	if (sid < 0) {
		// Log any failures here
		goPrintf("EXIT_FAILURE 2");
		exit(EXIT_FAILURE);
	}
	// Close out the standard file descriptors
	for( int fd = 0; fd < 3; fd ++ ){
		close(fd);
	}
	// Change the current working directory
	if ((chdir("./")) < 0) {
		goPrintf("EXIT_FAILURE 3");
		exit(EXIT_FAILURE);
	}

	sfn = fn;
	return fds[1];
}

void EnterDaemon(int fd, int err)
{
	int n;
	daemon_info ret;
	ret.err = err;
	ret.pid = getpid();
	write(fd, &ret, sizeof(ret) );
}
*/
import "C"
import "unsafe"

func BeginDaemon(args []string) (C.int) {
	if len(args) > 1 && args[1] == "-d" {
		return 0
	}

	fnC := C.CString(args[0])
	defer C.free(unsafe.Pointer(fnC))
	fd := C.beginDaemon(fnC)
	return fd
}

func EnterDaemon(fd, err C.int) {
	C.EnterDaemon(fd, err)
}

