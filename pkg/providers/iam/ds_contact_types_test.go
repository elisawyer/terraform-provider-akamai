package iam

import (
	"testing"

	"github.com/akamai/terraform-provider-akamai/v2/pkg/test"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/mock"
)

func TestDSContactTypes(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		client := &IAM{}
		client.Test(test.TattleT{T: t})
		client.On("SupportedContactTypes", mock.Anything).Return([]string{"first", "second", "third"}, nil)

		p := provider{}
		p.SetCache(metaCache{})
		p.SetIAM(client)

		resource.UnitTest(t, resource.TestCase{
			ProviderFactories: p.ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: test.Fixture("testdata/%s/step0.tf", t.Name()),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.akamai_iam_contact_types.test", "id"),
						resource.TestCheckResourceAttr("data.akamai_iam_contact_types.test", "contact_types.0", "first"),
						resource.TestCheckResourceAttr("data.akamai_iam_contact_types.test", "contact_types.1", "second"),
						resource.TestCheckResourceAttr("data.akamai_iam_contact_types.test", "contact_types.2", "third"),
					),
				},
			},
		})

		client.AssertExpectations(t)
	})
}
