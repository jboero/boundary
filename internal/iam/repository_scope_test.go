package iam

import (
	"context"
	"testing"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/watchtower/internal/db"
	"github.com/hashicorp/watchtower/internal/oplog"
	"github.com/stretchr/testify/assert"
)

func Test_Repository_CreateScope(t *testing.T) {
	t.Parallel()
	cleanup, conn, _ := db.TestSetup(t, "postgres")
	defer cleanup()
	assert := assert.New(t)
	defer conn.Close()

	t.Run("valid-scope", func(t *testing.T) {
		rw := db.New(conn)
		wrapper := db.TestWrapper(t)
		repo, err := NewRepository(rw, rw, wrapper)
		id, err := uuid.GenerateUUID()
		assert.Nil(err)

		s, err := NewOrganization(WithName("fname-" + id))
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		foundScope, err := repo.LookupScope(context.Background(), WithPublicId(s.PublicId))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		foundScope, err = repo.LookupScope(context.Background(), WithName("fname-"+id))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		err = TestVerifyOplog(rw, s.PublicId)
		assert.Nil(err)
	})
	t.Run("dup-org-names", func(t *testing.T) {
		rw := db.New(conn)
		wrapper := db.TestWrapper(t)
		repo, err := NewRepository(rw, rw, wrapper)
		id, err := uuid.GenerateUUID()
		assert.Nil(err)

		s, err := NewOrganization(WithName("fname-" + id))
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		s2, err := NewOrganization(WithName("fname-" + id))
		s2, err = repo.CreateScope(context.Background(), s2)
		assert.True(err != nil)
		assert.True(s2 == nil)
	})
	t.Run("dup-proj-names", func(t *testing.T) {
		rw := db.New(conn)
		wrapper := db.TestWrapper(t)
		repo, err := NewRepository(rw, rw, wrapper)
		id, err := uuid.GenerateUUID()
		assert.Nil(err)

		s, err := NewOrganization(WithName("fname-" + id))
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		p, err := NewProject(s.PublicId, WithName("fname-"+id))
		p, err = repo.CreateScope(context.Background(), p)
		assert.Nil(err)
		assert.True(p.PublicId != "")

		p2, err := NewProject(s.PublicId, WithName("fname-"+id))
		p2, err = repo.CreateScope(context.Background(), p2)
		assert.True(err != nil)
		assert.True(p2 == nil)
	})
}

func Test_Repository_UpdateScope(t *testing.T) {
	t.Parallel()
	cleanup, conn, _ := db.TestSetup(t, "postgres")
	defer cleanup()
	assert := assert.New(t)
	defer conn.Close()

	t.Run("valid-scope", func(t *testing.T) {
		rw := db.New(conn)
		wrapper := db.TestWrapper(t)
		repo, err := NewRepository(rw, rw, wrapper)
		id, err := uuid.GenerateUUID()
		assert.Nil(err)

		s, err := NewOrganization(WithName("fname-" + id))
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		foundScope, err := repo.LookupScope(context.Background(), WithPublicId(s.PublicId))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		foundScope, err = repo.LookupScope(context.Background(), WithName("fname-"+id))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		assert.Nil(err)
		err = TestVerifyOplog(rw, s.PublicId, WithOperation(oplog.OpType_OP_TYPE_CREATE), WithCreateNbf(10))
		assert.Nil(err)

		s.Name = "fname-" + id
		s.Description = "desc-id" // not in the field mask paths
		s, updatedRows, err := repo.UpdateScope(context.Background(), s, []string{"Name"})
		assert.Nil(err)
		assert.Equal(1, updatedRows)
		assert.True(s != nil)
		assert.Equal(s.GetName(), "fname-"+id)
		assert.Equal(foundScope.GetDescription(), "") // should  be "" after update in db

		foundScope, err = repo.LookupScope(context.Background(), WithName("fname-"+id))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())
		assert.Equal(foundScope.GetDescription(), "")

		err = TestVerifyOplog(rw, s.PublicId, WithOperation(oplog.OpType_OP_TYPE_UPDATE), WithCreateNbf(10))
		assert.Nil(err)
	})
	t.Run("bad-parent-scope", func(t *testing.T) {
		rw := db.New(conn)
		wrapper := db.TestWrapper(t)
		repo, err := NewRepository(rw, rw, wrapper)
		id, err := uuid.GenerateUUID()
		assert.Nil(err)

		s, err := NewOrganization(WithName("fname-" + id))
		assert.Nil(err)
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		project, err := NewProject(s.PublicId)
		assert.Nil(err)
		project, err = repo.CreateScope(context.Background(), project)
		assert.Nil(err)
		assert.True(project != nil)

		project.ParentId = project.PublicId
		project, updatedRows, err := repo.UpdateScope(context.Background(), project, []string{"ParentId"})
		assert.True(err != nil)
		assert.Equal(0, updatedRows)
		assert.Equal("failed to update scope: error on update you cannot change a scope's parent", err.Error())
	})
}

func Test_Repository_LookupScope(t *testing.T) {
	t.Parallel()
	cleanup, conn, _ := db.TestSetup(t, "postgres")
	defer cleanup()
	assert := assert.New(t)
	defer conn.Close()

	t.Run("found-and-not-found", func(t *testing.T) {
		rw := db.New(conn)
		wrapper := db.TestWrapper(t)
		repo, err := NewRepository(rw, rw, wrapper)
		id, err := uuid.GenerateUUID()
		assert.Nil(err)

		s, err := NewOrganization(WithName("fname-" + id))
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		foundScope, err := repo.LookupScope(context.Background(), WithPublicId(s.PublicId))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		foundScope, err = repo.LookupScope(context.Background(), WithName("fname-"+id))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		invalidId, err := uuid.GenerateUUID()
		assert.Nil(err)
		notFoundById, err := repo.LookupScope(context.Background(), WithPublicId(invalidId))
		assert.Nil(err)
		assert.Nil(notFoundById)

		notFoundByName, err := repo.LookupScope(context.Background(), WithName(invalidId))
		assert.Nil(err)
		assert.Nil(notFoundByName)
	})
}

func Test_Repository_DeleteScope(t *testing.T) {
	t.Parallel()
	cleanup, conn, _ := db.TestSetup(t, "postgres")
	defer cleanup()
	assert := assert.New(t)
	defer conn.Close()
	rw := db.New(conn)
	wrapper := db.TestWrapper(t)
	repo, err := NewRepository(rw, rw, wrapper)
	assert.Nil(err)

	t.Run("valid-with-public-id", func(t *testing.T) {
		id, err := uuid.GenerateUUID()
		assert.Nil(err)
		s, err := NewOrganization(WithName("fname-" + id))
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		foundScope, err := repo.LookupScope(context.Background(), WithPublicId(s.PublicId))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		rowsDeleted, err := repo.DeleteScope(context.Background(), WithPublicId(s.PublicId))
		assert.Nil(err)
		assert.Equal(1, rowsDeleted)
		foundScope, err = repo.LookupScope(context.Background(), WithPublicId(s.PublicId))
		assert.Nil(err)
		assert.Nil(foundScope)

	})
	t.Run("valid-with-name", func(t *testing.T) {
		id, err := uuid.GenerateUUID()
		assert.Nil(err)
		s, err := NewOrganization(WithName("fname-" + id))
		s, err = repo.CreateScope(context.Background(), s)
		assert.Nil(err)
		assert.True(s != nil)
		assert.True(s.GetPublicId() != "")
		assert.Equal(s.GetName(), "fname-"+id)

		foundScope, err := repo.LookupScope(context.Background(), WithName("fname-"+id))
		assert.Nil(err)
		assert.Equal(foundScope.GetPublicId(), s.GetPublicId())

		rowsDeleted, err := repo.DeleteScope(context.Background(), WithName(s.Name))
		assert.Nil(err)
		assert.Equal(1, rowsDeleted)

		foundScope, err = repo.LookupScope(context.Background(), WithPublicId(s.PublicId))
		assert.Nil(err)
		assert.Nil(foundScope)
	})
	t.Run("valid-with-bad-id-or-name", func(t *testing.T) {
		invalidId, err := uuid.GenerateUUID()
		foundScope, err := repo.LookupScope(context.Background(), WithPublicId(invalidId))
		assert.Nil(err)
		assert.Nil(foundScope)
		assert.Nil(err)
		rowsDeleted, err := repo.DeleteScope(context.Background(), WithPublicId(invalidId))
		assert.Nil(err) // no error is expected if the resource isn't in the db
		assert.Equal(0, rowsDeleted)

		rowsDeleted, err = repo.DeleteScope(context.Background(), WithName(invalidId))
		assert.Nil(err) // no error is expected if the resource isn't in the db
		assert.Equal(0, rowsDeleted)
	})
}
