package instorage

/*
import (
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"inspur.com/storage/instorage-k8s/pkg/tests/mock_ssh"
)


func TestCreateVolumeFailedSSHFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	mockSSH.EXPECT().Execute(gomock.Any()).Return("", "argument invalid", 0, fmt.Errorf("argument invalid"))
	expectErrStr := "create object failed argument invalid"

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"
	options["deviceUsername"] = "superuser"
	options["devicePassword"] = "00000000"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volThin] = "true"
	volCreateOptions[volThinResize] = "20"
	volCreateOptions[volThinWarning] = "80"
	volCreateOptions[volThinGrainSize] = "256"

	_, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if err.Error() != expectErrStr {
		t.Errorf("volume create should failed for '%s', actually failed for '%s'", expectErrStr, err)
	}
}

func TestCreateVolumeFailedObjectNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	mockSSH.EXPECT().Execute(gomock.Any()).Return("something unable to understand", "", 0, nil)
	expectErrStr := "object ID not found, response parse failed"

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volThin] = "true"
	volCreateOptions[volThinResize] = "20"
	volCreateOptions[volThinWarning] = "80"
	volCreateOptions[volThinGrainSize] = "256"

	_, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if err.Error() != expectErrStr {
		t.Errorf("volume create should failed for '%s', actually failed for '%s'", expectErrStr, err)
	}
}

func TestBuildVolumeCreateParameterThinProvisionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvdisk -name vol-001 -size 10 -unit gb -mdiskgrp Pool0 -iogrp 0 -rsize 20% -warning 80% -autoexpand -grainsize 256 -intier off"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volIOGrp] = "0"
	volCreateOptions[volThin] = "true"
	volCreateOptions[volThinResize] = "20"
	volCreateOptions[volThinWarning] = "80"
	volCreateOptions[volThinGrainSize] = "256"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterCompressSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvdisk -name vol-001 -size 10 -unit gb -mdiskgrp Pool0 -iogrp 0 -rsize 2% -autoexpand -compressed -intier off"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volIOGrp] = "0"
	volCreateOptions[volCompress] = "true"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterInTierSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvdisk -name vol-001 -size 10 -unit gb -mdiskgrp Pool0 -iogrp 0 -intier on"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volIOGrp] = "0"
	volCreateOptions[volInTier] = "true"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterThickProvisionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvdisk -name vol-001 -size 10 -unit gb -mdiskgrp Pool0 -iogrp 0 -intier off"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volIOGrp] = "0"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterMirrorSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvdisk -name vol-001 -size 10 -unit gb -copies 2 -mdiskgrp Pool0:Pool1 -accessiogrp 0 -iogrp 0 -intier off"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volAuxPoolName] = "Pool1"
	volCreateOptions[volIOGrp] = "0"
	volCreateOptions[volLevel] = "mirror"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterAAVolumeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvolume -name vol-001 -size 10 -unit gb -pool Pool0:Pool1 -iogrp 0:1"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volAuxPoolName] = "Pool1"
	volCreateOptions[volIOGrp] = "0"
	volCreateOptions[volAuxIOGrp] = "1"
	volCreateOptions[volLevel] = "aa"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterAAThinVolumeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvolume -name vol-001 -size 10 -unit gb -pool Pool0:Pool1 -iogrp 0:1 -buffersize 20% -warning 80% -thin -grainsize 256"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volAuxPoolName] = "Pool1"
	volCreateOptions[volIOGrp] = "0"
	volCreateOptions[volAuxIOGrp] = "1"
	volCreateOptions[volLevel] = "aa"
	volCreateOptions[volThin] = "true"
	volCreateOptions[volThinResize] = "20"
	volCreateOptions[volThinWarning] = "80"
	volCreateOptions[volThinGrainSize] = "256"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterAACompressVolumeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	expectArg := "mcsop mkvolume -name vol-001 -size 10 -unit gb -pool Pool0:Pool1 -iogrp 0:1 -buffersize 2% -compressed"
	mockSSH.EXPECT().Execute(expectArg).Return("Virtual Disk, id [12345], successfully created", "", 0, nil)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	//make test volume create option
	volCreateOptions := make(map[string]string)
	volCreateOptions[volPoolName] = "Pool0"
	volCreateOptions[volAuxPoolName] = "Pool1"
	volCreateOptions[volIOGrp] = "0"
	volCreateOptions[volAuxIOGrp] = "1"
	volCreateOptions[volLevel] = "aa"
	volCreateOptions[volCompress] = "true"

	id, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
	if id != "12345" {
		t.Errorf("volume id should be 12345 actually %s", id)
	}
	if err != nil {
		t.Errorf("volume create should success, actually failed for %s", err)
	}
}

func TestBuildVolumeCreateParameterFailure(t *testing.T) {
	name1 := "PoolNotSet"
	options1 := make(map[string]string)
	//options1[volPoolName] = "Pool0"
	options1[volIOGrp] = "0"
	options1[volThin] = "true"
	options1[volThinResize] = "20"
	options1[volThinWarning] = "80"
	options1[volThinGrainSize] = "64"
	expect1 := "Pool should be set when create volume."

	name2 := "ResizeNotInteger"
	options2 := make(map[string]string)
	options2[volPoolName] = "Pool0"
	options2[volIOGrp] = "0"
	options2[volThin] = "true"
	options2[volThinResize] = "abc"
	options2[volThinWarning] = "80"
	options2[volThinGrainSize] = "64"
	expect2 := "resize should be integer"

	name3 := "ResizeTooLarge"
	options3 := make(map[string]string)
	options3[volPoolName] = "Pool0"
	options3[volIOGrp] = "0"
	options3[volThin] = "true"
	options3[volThinResize] = "101"
	options3[volThinWarning] = "80"
	options3[volThinGrainSize] = "64"
	expect3 := "resize should be in the range (0, 100]"

	name4 := "ResizeTooSmall"
	options4 := make(map[string]string)
	options4[volPoolName] = "Pool0"
	options4[volIOGrp] = "0"
	options4[volThin] = "true"
	options4[volThinResize] = "0"
	options4[volThinWarning] = "80"
	options4[volThinGrainSize] = "64"
	expect4 := "resize should be in the range (0, 100]"

	name5 := "WarningNotInteger"
	options5 := make(map[string]string)
	options5[volPoolName] = "Pool0"
	options5[volIOGrp] = "0"
	options5[volThin] = "true"
	options5[volThinResize] = "20"
	options5[volThinWarning] = "abc"
	options5[volThinGrainSize] = "64"
	expect5 := "warning should be integer"

	name6 := "WarningTooSmall"
	options6 := make(map[string]string)
	options6[volPoolName] = "Pool0"
	options6[volIOGrp] = "0"
	options6[volThin] = "true"
	options6[volThinResize] = "20"
	options6[volThinWarning] = "0"
	options6[volThinGrainSize] = "64"
	expect6 := "warning should be in the range (0, 100)"

	name7 := "WarningTooLarge"
	options7 := make(map[string]string)
	options7[volPoolName] = "Pool0"
	options7[volIOGrp] = "0"
	options7[volThin] = "true"
	options7[volThinResize] = "20"
	options7[volThinWarning] = "101"
	options7[volThinGrainSize] = "64"
	expect7 := "warning should be in the range (0, 100)"

	name8 := "GrainSizeNotValid"
	options8 := make(map[string]string)
	options8[volPoolName] = "Pool0"
	options8[volIOGrp] = "0"
	options8[volThin] = "true"
	options8[volThinResize] = "20"
	options8[volThinWarning] = "80"
	options8[volThinGrainSize] = "15"
	expect8 := "Thin GrainSize can only be 32 or 64 or 128 or 256."

	for _, data := range []struct {
		name   string
		option map[string]string
		expect string
	}{
		{name1, options1, expect1},
		{name2, options2, expect2},
		{name3, options3, expect3},
		{name4, options4, expect4},
		{name5, options5, expect5},
		{name6, options6, expect6},
		{name7, options7, expect7},
		{name8, options8, expect8},
	} {
		t.Run(data.name, func(volCreateOptions map[string]string, expect string) func(*testing.T) {
			return func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockSSH := mock_ssh.NewMockIExecutor(ctrl)

				options := make(map[string]string)
				options["host"] = "1.1.1.1:22"
				options["login"] = "root"
				options["password"] = "password"

				strUtil := NewStorageUtil(options)
				//mock the ssh executor
				strUtil.cliWrapper.sshExecutor = mockSSH

				_, err := strUtil.CreateVolume("vol-001", "10", volCreateOptions)
				if err.Error() != expect {
					t.Errorf("Volume create should failed for '%s', actually '%s'", expect, err)
				}
			}
		}(data.option, data.expect))
	}
}

func TestDeleteVolumeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	lsvdiskExpect := "mcsinq lsvdisk -bytes -delim ! -filtervalue volume_name=fake-vol-name"
	lsvdiskReturn := "id!name!IO_group_id!IO_group_name!status!mdisk_grp_id!mdisk_grp_name!capacity!type!LC_id!LC_name!RC_id!RC_name!vdisk_UID!lc_map_count!copy_count!fast_write_state!se_copy_count!RC_change!compressed_copy_count!parent_mdisk_grp_id!parent_mdisk_grp_name!formatting!encrypt!volume_id!volume_name!function!ica!ica_bypass!ica_pid\n8!fake-vol-name!0!io_grp0!online!0!Pool0!1073741824!striped!!!!!60050760008B09C0D000000000008062!0!1!empty!0!no!0!0!Pool0!no!no!8!test-v1!!off!off!\n"
	rmvdiskExpect := "mcsop rmvdisk -force fake-vol-name"
	gomock.InOrder(
		//lsvdisk success
		mockSSH.EXPECT().Execute(lsvdiskExpect).Return(lsvdiskReturn, "", 0, nil),
		//rmvdisk success
		mockSSH.EXPECT().Execute(rmvdiskExpect).Return("", "", 0, nil),
	)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	err := strUtil.DeleteVolume("fake-vol-name")
	if err != nil {
		t.Errorf("volume should delete success, actually failed for %s", err)
	}
}

func TestDeleteVolumeFailed(t *testing.T) {
	for _, data := range []struct {
		stdout string
		stderr string
		code   int
		err    error

		expectSubinfo string
	}{
		{"", "", 0, fmt.Errorf("ssh failed"), "ssh failed"},
		{"", "some unexpected error output", 0, nil, "unexpected output from cmd"},
	} {
		t.Run("name", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSSH := mock_ssh.NewMockIExecutor(ctrl)
			mockSSH.EXPECT().Execute(gomock.Any()).Return(data.stdout, data.stderr, data.code, data.err).Times(1)

			options := make(map[string]string)
			options["host"] = "1.1.1.1:22"
			options["login"] = "root"
			options["password"] = "password"

			strUtil := NewStorageUtil(options)
			//mock the ssh executor
			strUtil.cliWrapper.sshExecutor = mockSSH

			err := strUtil.DeleteVolume("fake-vol-name")
			if !strings.Contains(err.Error(), data.expectSubinfo) {
				t.Errorf("not contain expected info %s, actualy %s", data.expectSubinfo, err)
			}
		})
	}
}

func TestDeleteVolumeFailedLsvdiskFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	lsvdiskExpect := "mcsinq lsvdisk -bytes -delim ! -filtervalue volume_name=fake-vol-name"
	rmvdiskExpect := "mcsop rmvdisk -force fake-vol-name"
	gomock.InOrder(
		//lsvdisk failed
		mockSSH.EXPECT().Execute(lsvdiskExpect).Return("", "", 0, fmt.Errorf("ssh failed")).Times(1),
		//rmvdisk failed
		mockSSH.EXPECT().Execute(rmvdiskExpect).Return("", "", 0, fmt.Errorf("ssh failed")).Times(0),
	)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	err := strUtil.DeleteVolume("fake-vol-name")
	if err.Error() != "ssh failed" {
		t.Errorf("volume should delete failed for ssh failed, actually failed for '%s'", err)
	}
}

func TestDeleteVolumeFailedRmvdiskSSHFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	lsvdiskExpect := "mcsinq lsvdisk -bytes -delim ! -filtervalue volume_name=fake-vol-name"
	lsvdiskReturn := "id!name!IO_group_id!IO_group_name!status!mdisk_grp_id!mdisk_grp_name!capacity!type!LC_id!LC_name!RC_id!RC_name!vdisk_UID!lc_map_count!copy_count!fast_write_state!se_copy_count!RC_change!compressed_copy_count!parent_mdisk_grp_id!parent_mdisk_grp_name!formatting!encrypt!volume_id!volume_name!function!ica!ica_bypass!ica_pid\n8!fake-vol-name!0!io_grp0!online!0!Pool0!1073741824!striped!!!!!60050760008B09C0D000000000008062!0!1!empty!0!no!0!0!Pool0!no!no!8!test-v1!!off!off!\n"
	rmvdiskExpect := "mcsop rmvdisk -force fake-vol-name"
	gomock.InOrder(
		//lsvdisk failed
		mockSSH.EXPECT().Execute(lsvdiskExpect).Return(lsvdiskReturn, "", 0, nil).Times(1),
		//rmvdisk failed
		mockSSH.EXPECT().Execute(rmvdiskExpect).Return("", "", 0, fmt.Errorf("ssh failed")).Times(1),
	)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	err := strUtil.DeleteVolume("fake-vol-name")
	if err.Error() != "ssh failed" {
		t.Errorf("volume should delete failed for ssh failed, actually failed for '%s'", err)
	}
}

func TestDeleteVolumeFailedRmvdiskCmdFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSSH := mock_ssh.NewMockIExecutor(ctrl)
	lsvdiskExpect := "mcsinq lsvdisk -bytes -delim ! -filtervalue volume_name=fake-vol-name"
	lsvdiskReturn := "id!name!IO_group_id!IO_group_name!status!mdisk_grp_id!mdisk_grp_name!capacity!type!LC_id!LC_name!RC_id!RC_name!vdisk_UID!lc_map_count!copy_count!fast_write_state!se_copy_count!RC_change!compressed_copy_count!parent_mdisk_grp_id!parent_mdisk_grp_name!formatting!encrypt!volume_id!volume_name!function!ica!ica_bypass!ica_pid\n8!fake-vol-name!0!io_grp0!online!0!Pool0!1073741824!striped!!!!!60050760008B09C0D000000000008062!0!1!empty!0!no!0!0!Pool0!no!no!8!test-v1!!off!off!\n"
	rmvdiskExpect := "mcsop rmvdisk -force fake-vol-name"
	gomock.InOrder(
		//lsvdisk failed
		mockSSH.EXPECT().Execute(lsvdiskExpect).Return(lsvdiskReturn, "", 0, nil).Times(1),
		//rmvdisk failed
		mockSSH.EXPECT().Execute(rmvdiskExpect).Return("unexpected stdout output", "unexpected stderr output", 0, nil).Times(1),
	)

	options := make(map[string]string)
	options["host"] = "1.1.1.1:22"
	options["login"] = "root"
	options["password"] = "password"

	strUtil := NewStorageUtil(options)
	//mock the ssh executor
	strUtil.cliWrapper.sshExecutor = mockSSH

	err := strUtil.DeleteVolume("fake-vol-name")
	if !strings.Contains(err.Error(), "expected no output from command") {
		t.Errorf("volume should delete failed for unexpected stdout/stderr output, actually is '%s'", err)
	}
}
*/
