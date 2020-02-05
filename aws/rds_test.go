package aws

import (
	"strings"
	"testing"
	"time"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func waitUntilRdsCreated(svc *rds.RDS, name *string) error {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: name,
	}

	for i := 0; i < 240; i++ {
		instance, err := svc.DescribeDBInstances(input)
		status := instance.DBInstances[0].DBInstanceStatus

		// If SkipFinalSnapshot = false on delete, should also wait for "backing-up" also to finish
		if *status != "creating" {
			return nil
		}

		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
		logging.Logger.Debug("Waiting for RDS to be created")
	}

	return RdsDeleteError{}
}

func createTestRDSInstance(t *testing.T, session *session.Session, name string) {
	svc := rds.New(session)
	params := &rds.CreateDBInstanceInput{
		AllocatedStorage:     awsgo.Int64(5),
		DBInstanceClass:      awsgo.String("db.m5.large"),
		DBInstanceIdentifier: awsgo.String(name),
		Engine:               awsgo.String("postgres"),
		MasterUsername:       awsgo.String("gruntwork"),
		MasterUserPassword:   awsgo.String("password"),
	}

	_, err := svc.CreateDBInstance(params)
	require.NoError(t, err)

	waitUntilRdsCreated(svc, &name)
}

func TestListRDS(t *testing.T) {
	t.Parallel()

	region := "us-east-1" //, err := getRandomRegion()
	//if err != nil {
	//assert.Fail(t, errors.WithStackTrace(err).Error())
	//}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	rdsName := "cloud-nuke-test" + util.UniqueID()
	createTestRDSInstance(t, session, rdsName)

	eds, err := getAllRdsInstances(session, region, time.Now())

	if err != nil {
		assert.Failf(t, "Unable to fetch list of RDS", errors.WithStackTrace(err).Error())
	}

	assert.Contains(t, awsgo.StringValueSlice(eds), strings.ToLower(rdsName))

	nukeAllRdsInstances(session, eds)

	rdsNames, err := getAllRdsInstances(session, region, time.Now().Add(1*time.Hour*-1))

	assert.NotContains(t, awsgo.StringValueSlice(rdsNames), strings.ToLower(rdsName))
}
