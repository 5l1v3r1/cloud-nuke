package aws

import (
	"testing"

	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func createTestEBSVolume(t *testing.T, session *session.Session, name string) ec2.Volume {
	svc := ec2.New(session)
	volume, err := svc.CreateVolume(&ec2.CreateVolumeInput{
		AvailabilityZone: awsgo.String("us-west-2a"),
		Size:             awsgo.Int64(8),
	})

	if err != nil {
		assert.Failf(t, "Could not create test EBS volume: %s", errors.WithStackTrace(err).Error())
	}

	err = svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: []*string{awsgo.String(*volume.VolumeId)},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	// Add test tag to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{volume.VolumeId},
		Tags: []*ec2.Tag{
			{
				Key:   awsgo.String("Name"),
				Value: awsgo.String(name),
			},
		},
	})

	if err != nil {
		assert.Failf(t, "Could not tag EBS volume: %s", errors.WithStackTrace(err).Error())
	}

	return *volume
}

func findEBSVolumesByNameTag(output *ec2.DescribeVolumesOutput, name string) []*string {
	var volumeIds []*string
	for _, volume := range output.Volumes {
		// Retrive only IDs of instances with the unique test tag
		for _, tag := range volume.Tags {
			if *tag.Key == "Name" {
				if *tag.Value == name {
					volumeIds = append(volumeIds, volume.VolumeId)
				}
			}
		}
	}

	return volumeIds
}

func TestListEBSVolumes(t *testing.T) {
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String("us-west-2")},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + uniqueID()
	volume := createTestEBSVolume(t, session, uniqueTestID)

	volumeIds, err := getAllEbsVolumes(session, "us-west-2")
	if err != nil {
		assert.Fail(t, "Unable to fetch list of EBS Volumes")
	}

	assert.Contains(t, awsgo.StringValueSlice(volumeIds), awsgo.StringValue(volume.VolumeId))
}

func TestNukeEBSVolumes(t *testing.T) {
	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String("us-west-2")},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "aws-nuke-test-" + uniqueID()
	createTestEC2Instance(t, session, uniqueTestID)

	output, err := ec2.New(session).DescribeVolumes(&ec2.DescribeVolumesInput{})
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	volumeIds := findEBSVolumesByNameTag(output, uniqueTestID)

	if err := nukeAllEbsVolumes(session, volumeIds); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
	volumes, err := getAllEbsVolumes(session, "us-west-2")

	if err != nil {
		assert.Fail(t, "Unable to fetch list of EC2 Instances")
	}

	for _, volumeID := range volumeIds {
		assert.NotContains(t, volumes, *volumeID)
	}
}
