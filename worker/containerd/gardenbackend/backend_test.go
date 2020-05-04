package gardenbackend_test

import (
	"errors"
	"testing"
	"time"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/gardenfakes"
	"github.com/concourse/concourse/worker/containerd/gardenbackend"
	"github.com/concourse/concourse/worker/containerd/gardenbackend/gardenbackendfakes"
	"github.com/concourse/concourse/worker/containerd/containerdfakes"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BackendSuite struct {
	suite.Suite
	*require.Assertions

	backend gardenbackend.Backend
	client  *containerdfakes.FakeClient
	network *gardenbackendfakes.FakeNetwork
	userns  *gardenbackendfakes.FakeUserNamespace
	killer  *gardenbackendfakes.FakeKiller
}

func (s *BackendSuite) SetupTest() {
	s.client = new(containerdfakes.FakeClient)
	s.killer = new(gardenbackendfakes.FakeKiller)
	s.network = new(gardenbackendfakes.FakeNetwork)
	s.userns = new(gardenbackendfakes.FakeUserNamespace)

	var err error
	s.backend, err = gardenbackend.New(s.client,
		gardenbackend.WithKiller(s.killer),
		gardenbackend.WithNetwork(s.network),
		gardenbackend.WithUserNamespace(s.userns),
	)
	s.NoError(err)
}

func (s *BackendSuite) TestNew() {
	_, err := gardenbackend.New(nil)
	s.EqualError(err, "nil client")
}

func (s *BackendSuite) TestPing() {
	for _, tc := range []struct {
		desc          string
		versionReturn error
		succeeds      bool
	}{
		{
			desc:          "fail from containerd version service",
			succeeds:      true,
			versionReturn: nil,
		},
		{
			desc:          "ok from containerd's version service",
			succeeds:      false,
			versionReturn: errors.New("error returning version"),
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			s.client.VersionReturns(tc.versionReturn)

			err := s.backend.Ping()
			if tc.succeeds {
				s.NoError(err)
				return
			}

			s.EqualError(errors.Unwrap(err), "error returning version")
		})
	}
}

var (
	invalidGdnSpec      = garden.ContainerSpec{}
	minimumValidGdnSpec = garden.ContainerSpec{
		Handle: "handle", RootFSPath: "raw:///rootfs",
	}
)

func (s *BackendSuite) TestCreateWithInvalidSpec() {
	_, err := s.backend.Create(invalidGdnSpec)

	s.Error(err)
	s.Equal(0, s.client.NewContainerCallCount())
}

func (s *BackendSuite) TestCreateWithNewContainerFailure() {
	s.client.NewContainerReturns(nil, errors.New("err"))

	_, err := s.backend.Create(minimumValidGdnSpec)
	s.Error(err)

	s.Equal(1, s.client.NewContainerCallCount())
}

func (s *BackendSuite) TestCreateContainerNewTaskFailure() {
	fakeContainer := new(containerdfakes.FakeContainer)

	expectedErr := errors.New("task-err")
	fakeContainer.NewTaskReturns(nil, expectedErr)

	s.client.NewContainerReturns(fakeContainer, nil)

	_, err := s.backend.Create(minimumValidGdnSpec)
	s.EqualError(errors.Unwrap(err), expectedErr.Error())

	s.Equal(1, fakeContainer.NewTaskCallCount())
}

func (s *BackendSuite) TestCreateContainerTaskStartFailure() {
	fakeTask := new(containerdfakes.FakeTask)
	fakeContainer := new(containerdfakes.FakeContainer)

	s.client.NewContainerReturns(fakeContainer, nil)
	fakeContainer.NewTaskReturns(fakeTask, nil)
	fakeTask.StartReturns(errors.New("start-err"))

	_, err := s.backend.Create(minimumValidGdnSpec)
	s.Error(err)

	s.EqualError(errors.Unwrap(err), "start-err")
}

func (s *BackendSuite) TestCreateContainerSetsHandle() {
	fakeTask := new(containerdfakes.FakeTask)
	fakeContainer := new(containerdfakes.FakeContainer)

	fakeContainer.IDReturns("handle")
	fakeContainer.NewTaskReturns(fakeTask, nil)

	s.client.NewContainerReturns(fakeContainer, nil)
	cont, err := s.backend.Create(minimumValidGdnSpec)
	s.NoError(err)

	s.Equal("handle", cont.Handle())

}

func (s *BackendSuite) TestContainersWithContainerdFailure() {
	s.client.ContainersReturns(nil, errors.New("err"))

	_, err := s.backend.Containers(nil)
	s.Error(err)
	s.Equal(1, s.client.ContainersCallCount())
}

func (s *BackendSuite) TestContainersWithInvalidPropertyFilters() {
	for _, tc := range []struct {
		desc   string
		filter map[string]string
	}{
		{
			desc: "empty key",
			filter: map[string]string{
				"": "bar",
			},
		},
		{
			desc: "empty value",
			filter: map[string]string{
				"foo": "",
			},
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			_, err := s.backend.Containers(tc.filter)

			s.Error(err)
			s.Equal(0, s.client.ContainersCallCount())
		})
	}
}

func (s *BackendSuite) TestContainersWithProperProperties() {
	_, _ = s.backend.Containers(map[string]string{"foo": "bar", "caz": "zaz"})
	s.Equal(1, s.client.ContainersCallCount())

	_, labelSet := s.client.ContainersArgsForCall(0)
	s.ElementsMatch([]string{"labels.foo==bar", "labels.caz==zaz"}, labelSet)
}

func (s *BackendSuite) TestContainersConversion() {
	fakeContainer1 := new(containerdfakes.FakeContainer)
	fakeContainer2 := new(containerdfakes.FakeContainer)

	s.client.ContainersReturns([]containerd.Container{
		fakeContainer1, fakeContainer2,
	}, nil)

	containers, err := s.backend.Containers(nil)
	s.NoError(err)
	s.Equal(1, s.client.ContainersCallCount())
	s.Len(containers, 2)
}

func (s *BackendSuite) TestLookupEmptyHandleError() {
	_, err := s.backend.Lookup("")
	s.Equal("empty handle", err.Error())
}

func (s *BackendSuite) TestLookupCallGetContainerWithHandle() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeContainer.IDReturns("handle")
	s.client.GetContainerReturns(fakeContainer, nil)

	_, _ = s.backend.Lookup("handle")
	s.Equal(1, s.client.GetContainerCallCount())

	_, handle := s.client.GetContainerArgsForCall(0)
	s.Equal("handle", handle)
}

func (s *BackendSuite) TestLookupGetContainerError() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeContainer.IDReturns("handle")
	s.client.GetContainerReturns(fakeContainer, nil)

	s.client.GetContainerReturns(nil, errors.New("containerd-err"))

	_, err := s.backend.Lookup("handle")
	s.Error(err)
	s.EqualError(errors.Unwrap(err), "containerd-err")
}

func (s *BackendSuite) TestLookupGetContainerFails() {
	s.client.GetContainerReturns(nil, errors.New("err"))
	_, err := s.backend.Lookup("non-existent-handle")
	s.Error(err)
	s.EqualError(errors.Unwrap(err), "err")
}

func (s *BackendSuite) TestLookupGetNoContainerReturned() {
	s.client.GetContainerReturns(nil, errors.New("not found"))
	container, err := s.backend.Lookup("non-existent-handle")
	s.Error(err)
	s.Nil(container)
}

func (s *BackendSuite) TestLookupGetContainer() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeContainer.IDReturns("handle")
	s.client.GetContainerReturns(fakeContainer, nil)
	container, err := s.backend.Lookup("handle")
	s.NoError(err)
	s.NotNil(container)
	s.Equal("handle", container.Handle())
}

func (s *BackendSuite) TestDestroyEmptyHandleError() {
	err := s.backend.Destroy("")
	s.EqualError(err, "empty handle")
}

func (s *BackendSuite) TestDestroyGetContainerError() {
	s.client.GetContainerReturns(nil, errors.New("get-container-failed"))

	err := s.backend.Destroy("some-handle")
	s.EqualError(errors.Unwrap(err), "get-container-failed")
}

func (s *BackendSuite) TestDestroyGetTaskError() {
	fakeContainer := new(containerdfakes.FakeContainer)

	s.client.GetContainerReturns(fakeContainer, nil)

	expectedError := errors.New("get-task-failed")
	fakeContainer.TaskReturns(nil, expectedError)

	err := s.backend.Destroy("some handle")
	s.True(errors.Is(err, expectedError))
}

func (s *BackendSuite) TestDestroyGetTaskErrorNotFoundAndDeleteFails() {
	fakeContainer := new(containerdfakes.FakeContainer)

	s.client.GetContainerReturns(fakeContainer, nil)
	fakeContainer.TaskReturns(nil, errdefs.ErrNotFound)

	expectedError := errors.New("delete-container-failed")
	fakeContainer.DeleteReturns(expectedError)

	err := s.backend.Destroy("some handle")
	s.True(errors.Is(err, expectedError))
}

func (s *BackendSuite) TestDestroyGetTaskErrorNotFoundAndDeleteSucceeds() {
	fakeContainer := new(containerdfakes.FakeContainer)

	s.client.GetContainerReturns(fakeContainer, nil)
	fakeContainer.TaskReturns(nil, errdefs.ErrNotFound)

	err := s.backend.Destroy("some handle")

	s.Equal(1, fakeContainer.DeleteCallCount())
	s.NoError(err)
}

func (s *BackendSuite) TestDestroyKillTaskFails() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeTask := new(containerdfakes.FakeTask)

	s.client.GetContainerReturns(fakeContainer, nil)
	fakeContainer.TaskReturns(fakeTask, nil)

	expectedError := errors.New("kill-task-failed")
	s.killer.KillReturns(expectedError)

	err := s.backend.Destroy("some handle")
	s.True(errors.Is(err, expectedError))
	_, _, behaviour := s.killer.KillArgsForCall(0)
	s.Equal(gardenbackend.KillGracefully, behaviour)
}

func (s *BackendSuite) TestDestroyRemoveNetworkFails() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeTask := new(containerdfakes.FakeTask)

	s.client.GetContainerReturns(fakeContainer, nil)
	fakeContainer.TaskReturns(fakeTask, nil)

	expectedError := errors.New("remove-network-failed")
	s.network.RemoveReturns(expectedError)

	err := s.backend.Destroy("some handle")
	s.True(errors.Is(err, expectedError))
}

func (s *BackendSuite) TestDestroyDeleteTaskFails() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeTask := new(containerdfakes.FakeTask)

	s.client.GetContainerReturns(fakeContainer, nil)
	fakeContainer.TaskReturns(fakeTask, nil)

	expectedError := errors.New("delete-task-failed")
	fakeTask.DeleteReturns(nil, expectedError)

	err := s.backend.Destroy("some handle")
	s.True(errors.Is(err, expectedError))
}

func (s *BackendSuite) TestDestroyContainerDeleteFailsAndDeleteTaskSucceeds() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeTask := new(containerdfakes.FakeTask)

	s.client.GetContainerReturns(fakeContainer, nil)
	fakeContainer.TaskReturns(fakeTask, nil)

	expectedError := errors.New("delete-container-failed")
	fakeContainer.DeleteReturns(expectedError)

	err := s.backend.Destroy("some handle")
	s.True(errors.Is(err, expectedError))
}

func (s *BackendSuite) TestDestroySucceeds() {
	fakeContainer := new(containerdfakes.FakeContainer)
	fakeTask := new(containerdfakes.FakeTask)
	s.client.GetContainerReturns(fakeContainer, nil)
	fakeContainer.TaskReturns(fakeTask, nil)

	err := s.backend.Destroy("some handle")
	s.NoError(err)
}

func (s *BackendSuite) TestStart() {
	err := s.backend.Start()
	s.NoError(err)
	s.Equal(1, s.client.InitCallCount())
}

func (s *BackendSuite) TestStartInitError() {
	s.client.InitReturns(errors.New("init failed"))
	err := s.backend.Start()
	s.EqualError(errors.Unwrap(err), "init failed")
}

func (s *BackendSuite) TestStop() {
	s.backend.Stop()
	s.Equal(1, s.client.StopCallCount())
}

func (s *BackendSuite) TestGraceTimeGetPropertyFails() {
	fakeContainer := new(gardenfakes.FakeContainer)
	fakeContainer.PropertyReturns("", errors.New("error"))
	result := s.backend.GraceTime(fakeContainer)
	s.Equal(time.Duration(0), result)
}

func (s *BackendSuite) TestGraceTimeInvalidInteger() {
	fakeContainer := new(gardenfakes.FakeContainer)
	fakeContainer.PropertyReturns("not a number", nil)
	result := s.backend.GraceTime(fakeContainer)
	s.Equal(time.Duration(0), result)
}

func (s *BackendSuite) TestGraceTimeReturnsDuration() {
	fakeContainer := new(gardenfakes.FakeContainer)
	fakeContainer.PropertyReturns("123", nil)
	result := s.backend.GraceTime(fakeContainer)
	s.Equal(time.Duration(123), result)
}