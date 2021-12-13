#!/bin/bash

# Copyright 2020 The OpenEBS Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Make sure we react to these signals by running stop() when we see them - for clean shutdown
# And then exiting
trap "stop; exit 0;" SIGTERM SIGINT

stop()
{
  # We're here because we've seen SIGTERM, likely via a Docker stop command or similar
  # Let's shutdown cleanly
  echo "SIGTERM caught, terminating NFS process(es)..."
  /usr/sbin/exportfs -uav
  /usr/sbin/rpc.nfsd 0
  pid1=`pidof rpc.nfsd`
  pid2=`pidof rpc.mountd`
  # For IPv6 bug:
  pid3=`pidof rpcbind`
  kill -TERM $pid1 $pid2 $pid3 > /dev/null 2>&1
  echo "Terminated."
  exit
}

get_nfs_args() {
  declare -n args=$1

  args=(--debug 8 --no-udp --no-nfs-version 2 --no-nfs-version 3)

  # here we are checking if variable exist and its value is not null
  if [ ! -z ${NFS_GRACE_TIME:+x} ]; then
    args+=( --grace-time ${NFS_GRACE_TIME})
  fi

  if [ ! -z ${NFS_LEASE_TIME:+x} ]; then
    args+=( --lease-time ${NFS_LEASE_TIME})
  fi
}

# Check if the SHARED_DIRECTORY variable is empty
if [ -z "${SHARED_DIRECTORY}" ]; then
  echo "The SHARED_DIRECTORY environment variable is unset or null, exiting..."
  exit 1
else
  echo "Writing SHARED_DIRECTORY to /etc/exports file"
  /bin/sed -i "s@{{SHARED_DIRECTORY}}@${SHARED_DIRECTORY}@g" /etc/exports
fi

# This is here to demonsrate how multiple directories can be shared. You
# would need a block like this for each extra share.
# Any additional shares MUST be subdirectories of the root directory specified
# by SHARED_DIRECTORY.

# Check if the SHARED_DIRECTORY_2 variable is empty
if [ ! -z "${SHARED_DIRECTORY_2}" ]; then
  echo "Writing SHARED_DIRECTORY_2 to /etc/exports file"
  echo "{{SHARED_DIRECTORY_2}} {{PERMITTED}}({{READ_ONLY}},{{SYNC}},no_subtree_check,no_auth_nlm,insecure,no_root_squash)" >> /etc/exports
  /bin/sed -i "s@{{SHARED_DIRECTORY_2}}@${SHARED_DIRECTORY_2}@g" /etc/exports
fi

# Check if the PERMITTED variable is empty
if [ -z "${PERMITTED}" ]; then
  echo "The PERMITTED environment variable is unset or null, defaulting to '*'."
  echo "This means any client can mount."
  /bin/sed -i "s/{{PERMITTED}}/*/g" /etc/exports
else
  echo "The PERMITTED environment variable is set."
  echo "The permitted clients are: ${PERMITTED}."
  /bin/sed -i "s/{{PERMITTED}}/"${PERMITTED}"/g" /etc/exports
fi

# Check if the READ_ONLY variable is set (rather than a null string) using parameter expansion
if [ -z ${READ_ONLY+y} ]; then
  echo "The READ_ONLY environment variable is unset or null, defaulting to 'rw'."
  echo "Clients have read/write access."
  /bin/sed -i "s/{{READ_ONLY}}/rw/g" /etc/exports
else
  echo "The READ_ONLY environment variable is set."
  echo "Clients will have read-only access."
  /bin/sed -i "s/{{READ_ONLY}}/ro/g" /etc/exports
fi

# Check if the SYNC variable is set (rather than a null string) using parameter expansion
if [ -z "${SYNC+y}" ]; then
  echo "The SYNC environment variable is unset or null, defaulting to 'async' mode".
  echo "Writes will not be immediately written to disk."
  /bin/sed -i "s/{{SYNC}}/async/g" /etc/exports
else
  echo "The SYNC environment variable is set, using 'sync' mode".
  echo "Writes will be immediately written to disk."
  /bin/sed -i "s/{{SYNC}}/sync/g" /etc/exports
fi

# Check if the CUSTOM_EXPORTS_CONFIG variable is set, and if it is, clear the
# /etc/exports file that's already present and replace it with the contents
# of the CUSTOM_EXPORTS_CONFIG variable

# START OF JAMES CODE
if [ ! -z "${CUSTOM_EXPORTS_CONFIG}" ]; then
  echo "CUSTOM_EXPORTS_CONFIG variable is set, clearing /etc/exports..."
  if [ -f "etc/exports" ]; then  
    /bin/rm etc/exports
  fi
  echo "Adding the contents of \$CUSTOM_EXPORTS_CONFIG to /etc/exports..."
  echo $CUSTOM_EXPORTS_CONFIG > /etc/exports
  echo "Addition complete." 
fi
# END OF JAMES CODE

# Partially set 'unofficial Bash Strict Mode' as described here: http://redsymbol.net/articles/unofficial-bash-strict-mode/
# We don't set -e because the pidof command returns an exit code of 1 when the specified process is not found
# We expect this at times and don't want the script to be terminated when it occurs
set -uo pipefail
IFS=$'\n\t'

# Modify the shared directory (${SHARED_DIRECTORY}) file user owner
# Does not support more than one shared directory
if [ -n "${FILEPERMISSIONS_UID}" ]; then
  # These variables will be used to handle errors
  UID_ERROR=""
  CHOWN_UID_ERROR=""
  # Validating input UID value
  # Errors if UID is not a decimal number
  targetUID=$(printf %d ${FILEPERMISSIONS_UID}) || UID_ERROR=$?
  if [ -n "${UID_ERROR}" ]; then
    echo "user change error: Invalid UID ${FILEPERMISSIONS_UID}"
    exit 1
  fi

  presentUID=$(stat ${SHARED_DIRECTORY} --printf=%u)

  # OnRootMismatch-like check
  if [ "$presentUID" -ne "$targetUID" ]; then
    chown -R $targetUID ${SHARED_DIRECTORY} || CHOWN_UID_ERROR=$?
    if [ -n "${CHOWN_UID_ERROR}" ]; then
      echo "user change error: Failed to change user owner of ${SHARED_DIRECTORY}"
      exit 1
    fi

    echo "chown user command succeeded"
  fi
fi

# Modify the shared directory (${SHARED_DIRECTORY}) file group owner
# Does not support more than one shared directory
if [ -n "${FILEPERMISSIONS_GID}" ]; then
  # These variables will be used to handle errors
  GID_ERROR=""
  CHOWN_GID_ERROR=""
  # Validating input GID value
  # Errors if GID is not a decimal number
  targetGID=$(printf %d ${FILEPERMISSIONS_GID}) || GID_ERROR=$?
  if [ -n "${GID_ERROR}" ]; then
    echo "group change error: Invalid GID ${FILEPERMISSIONS_GID}"
    exit 1
  fi

  presentGID=$(stat ${SHARED_DIRECTORY} --printf=%g)

  # OnRootMismatch-like check
  if [ "$presentGID" -ne "$targetGID" ]; then
    chown -R :${targetGID} ${SHARED_DIRECTORY} || CHOWN_GID_ERROR=$?
    if [ -n "${CHOWN_GID_ERROR}" ]; then
      echo "group change error: Failed to change group owner of ${SHARED_DIRECTORY}"
      exit 1
    fi

    echo "chown group command succeeded"
  fi
fi

# Modify the shared directory (${SHARED_DIRECTORY}) file permissions
# Does not support more than one shared directory
if [ -n "${FILEPERMISSIONS_MODE}" ]; then
  # These variables will be used to handle errors
  TEST_CHMOD_ERROR=""
  CHMOD_ERROR=""
  
  # 'chmod -c' output is a non-empty string if the file mode changes
  # The TEST_CHMOD_OUT variable is used to capture this string
  TEST_CHMOD_OUT=$(chmod ${FILEPERMISSIONS_MODE} ${SHARED_DIRECTORY} -c) || TEST_CHMOD_ERROR=$?
  # If the command fails, the specified mode is invalid
  if [ -n "${TEST_CHMOD_ERROR}" ]; then
    echo "mode change error: chmod test command failed. 'mode' value ${FILEPERMISSIONS_MODE} might be invalid"
    exit 1
  fi

  # If the TEST_CHMOD_OUT is not empty, then there is a root mismatch
  # (Similar to OnRootMismatch)
  # Thus a recursive chmod is issued if there is root mismatch
  # NOTE: This test won't work if we want to handle root mismatch in
  #       any other way than the execution of the recursive chmod
  if [ -n "${TEST_CHMOD_OUT}" ]; then
    chmod -R ${FILEPERMISSIONS_MODE} ${SHARED_DIRECTORY} || CHMOD_ERROR=$?
    if [ -n "${CHMOD_ERROR}" ]; then
      echo "mode change error: Failed to change file mode of ${SHARED_DIRECTORY}"
      exit 1
    fi

    echo "chmod command succeeded"
  fi
fi

# This loop runs till until we've started up successfully
while true; do

  # Check if NFS is running by recording it's PID (if it's not running $pid will be null):
  pid=`pidof rpc.mountd`

  # If $pid is null, do this to start or restart NFS:
  while [ -z "$pid" ]; do
    echo "Displaying /etc/exports contents:"
    cat /etc/exports
    echo ""

    # Normally only required if v3 will be used
    # But currently enabled to overcome an NFS bug around opening an IPv6 socket
    echo "Starting rpcbind..."
    /sbin/rpcbind -w
    echo "Displaying rpcbind status..."
    /sbin/rpcinfo

    # Only required if v3 will be used
    # /usr/sbin/rpc.idmapd
    # /usr/sbin/rpc.gssd -v
    # /usr/sbin/rpc.statd

    echo "Starting NFS in the background..."
    get_nfs_args nfs_args
    /usr/sbin/rpc.nfsd ${nfs_args[@]}
    echo "Exporting File System..."
    if /usr/sbin/exportfs -rv; then
      /usr/sbin/exportfs
    else
      echo "Export validation failed, exiting..."
      exit 1
    fi
    echo "Starting Mountd in the background..."These
    /usr/sbin/rpc.mountd --debug all --no-udp --no-nfs-version 2 --no-nfs-version 3
# --exports-file /etc/exports

    # Check if NFS is now running by recording it's PID (if it's not running $pid will be null):
    pid=`pidof rpc.mountd`

    # If $pid is null, startup failed; log the fact and sleep for 2s
    # We'll then automatically loop through and try again
    if [ -z "$pid" ]; then
      echo "Startup of NFS failed, sleeping for 2s, then retrying..."
      sleep 2
    fi

  done

  # Break this outer loop once we've started up successfully
  # Otherwise, we'll silently restart and Docker won't know
  echo "Startup successful."
  break

done

while true; do

  # Check if NFS is STILL running by recording it's PID (if it's not running $pid will be null):
  pid=`pidof rpc.mountd`
  # If it is not, lets kill our PID1 process (this script) by breaking out of this while loop:
  # This ensures Docker observes the failure and handles it as necessary
  if [ -z "$pid" ]; then
    echo "NFS has failed, exiting, so Docker can restart the container..."
    break
  fi

  # If it is, give the CPU a rest
  sleep 1

done

sleep 1
exit 1
