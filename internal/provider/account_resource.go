package provider

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &accountResource{}
	_ resource.ResourceWithConfigure   = &accountResource{}
	_ resource.ResourceWithImportState = &accountResource{}
	_ resource.ResourceWithModifyPlan  = &accountResource{}
)

// NewAccountResource is a helper function to simplify the provider implementation.
func NewAccountResource() resource.Resource {
	return &accountResource{}
}

// accountResource is the resource implementation.
type accountResource struct {
	client *xrpc.Client
}

type accountResourceModel struct {
	Did             types.String `tfsdk:"did"`
	Email           types.String `tfsdk:"email"`
	Handle          types.String `tfsdk:"handle"`
	InitialPassword types.String `tfsdk:"initial_password"`
	// TODO:
	//recoveryKey       types.String `tfsdk:"recovery_key"`

	// These don't make sense to manage via TF:
	//inviteCode
	//verificationCode
	//verificationPhone
}

// Metadata returns the resource type name.
func (l *accountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

// Schema defines the schema for the resource.
func (r *accountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Accounts. This resource requires the provider to be configured with a user password, not an app password.",
		Attributes: map[string]schema.Attribute{
			"did": schema.StringAttribute{
				MarkdownDescription: "Account's DID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The email of the account",
				Required:            true,
			},
			"handle": schema.StringAttribute{
				MarkdownDescription: "Requested handle for the account",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"initial_password": schema.StringAttribute{
				MarkdownDescription: "Initial account password. If not specified one will be generated and included in the Terraform output in plaintext.",
				Sensitive:           true,
				Optional:            true,
			},
		},
	}
}

func (l *accountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from a plan.
	var plan accountResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	generatedPassword := false
	password := plan.InitialPassword.ValueString()
	if password == "" {
		generatedPassword = true

		// generate a password similiar to how pdsadmin does it: https://github.com/bluesky-social/pds/blob/f054eefea58e6cddf17eda14a55ecf157c2e034e/pdsadmin/account.sh#L65
		length := 30
		bytes := make([]byte, length)
		_, err := rand.Read(bytes)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating account",
				"Failed to generate random initial password: "+err.Error(),
			)
			return
		}

		password = base64.URLEncoding.EncodeToString(bytes)
		if len(password) > length {
			password = password[:length]
		}
	}

	// Get server DID
	server, err := atproto.ServerDescribeServer(ctx, l.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating account",
			"Could not get server DID, unexpected error: "+err.Error(),
		)
		return
	}

	// Create an invite code
	createInviteCodeInput := &atproto.ServerCreateInviteCode_Input{
		UseCount: 1,
	}
	inviteCode, err := atproto.ServerCreateInviteCode(ctx, l.client, createInviteCodeInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating account",
			"Could not create invite code, unexpected error: "+err.Error(),
		)
		return
	}

	// Get server auth for create account
	authOutput, err := atproto.ServerGetServiceAuth(ctx, l.client, server.Did, time.Now().Unix()+60, "com.atproto.server.createAccount")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating account",
			"Could not get server auth, unexpected error: "+err.Error(),
		)
		return
	}

	// Ok to override there, this is a copy of the client
	l.client.Auth.AccessJwt = authOutput.Token
	l.client.Auth.RefreshJwt = authOutput.Token

	// Generate API request body from plan. Adapted from the account migratio script:
	// https://github.com/bluesky-social/indigo/blob/main/cmd/goat/account_migrate.go
	createRecordInput := atproto.ServerCreateAccount_Input{
		Handle:     plan.Handle.ValueString(),
		Email:      plan.Email.ValueStringPointer(),
		Password:   &password,
		InviteCode: &inviteCode.Code,
	}

	// Create new account.
	createOutput, err := atproto.ServerCreateAccount(ctx, l.client, &createRecordInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating account",
			"Could not create account, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values.
	plan.Did = types.StringValue(createOutput.Did)

	// Set state to fully populated data.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if generatedPassword {
		resp.Diagnostics.AddWarning(
			"Initial password created",
			"Generated initial password for account "+plan.Handle.ValueString()+": "+password,
		)
	}
}

// Read refreshes the Terraform state with the latest data.
func (l *accountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state.
	var state accountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	account, err := atproto.AdminGetAccountInfo(ctx, l.client, state.Did.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to retrieve account",
			"Could not retrieve the current state of the account, error: "+err.Error(),
		)
		return
	}

	state.Handle = types.StringValue(account.Handle)
	state.Email = types.StringValue(*account.Email)

	// Set refreshed state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (l *accountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get current state.
	var state accountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// update email, handle, or password
	resp.Diagnostics.AddError("Not implemented", "Update is not implemented for accounts")
	return

	/*
		// Generate API request body from plan.
		uri, err := syntax.ParseATURI(state.Uri.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid starter pack URI",
				"Could not parse Bluesky starter pack URI "+state.Uri.ValueString()+": "+err.Error(),
			)
			return
		}
		record, err := atproto.RepoGetRecord(ctx, l.client, "", uri.Collection().String(), uri.Authority().String(), uri.RecordKey().String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to retrieve starter pack",
				"Could not retrieve the current state of the starter pack "+state.Uri.ValueString()+": "+err.Error(),
			)
			return
		}
		pack, ok := record.Value.Val.(*bsky.GraphStarterpack)
		if !ok {
			resp.Diagnostics.AddError(
				"Failed to parse retrieved starter pack",
				"Could not cast the returned starter pack into the expected type",
			)
			return
		}

		pack.Name = state.Name.ValueString()
		pack.Description = state.Description.ValueStringPointer()
		pack.List = state.ListUri.ValueString()

		// Update existing list.
		putRecordInput := &atproto.RepoPutRecord_Input{
			Collection: uri.Collection().String(),
			Repo:       uri.Authority().String(),
			Rkey:       uri.RecordKey().String(),
			SwapRecord: record.Cid,
			Record: &util.LexiconTypeDecoder{
				Val: pack,
			},
		}
		_, err = atproto.RepoPutRecord(ctx, l.client, putRecordInput)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to update starter pack",
				"Could not update starter pack "+state.Uri.ValueString()+": "+err.Error(),
			)
			return
		}

		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	*/
}

// Delete deletes the resource and removes the Terraform state on success.
func (l *accountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state accountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteRequest := &atproto.AdminDeleteAccount_Input{
		Did: state.Did.ValueString(),
	}
	err := atproto.AdminDeleteAccount(ctx, l.client, deleteRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting account",
			"Could not delete account, error: "+err.Error(),
		)
	}
}

// Configure adds the provider configured client to the resource.
func (l *accountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*xrpc.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *xrpc.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	l.client = client
}

func (l *accountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import DID and save to did attribute.
	resource.ImportStatePassthroughID(ctx, path.Root("did"), req, resp)
}

func (l *accountResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() != req.State.Raw.IsNull() {
		// account is being created or deleted, require user password. App password will not work.
		claims := jwt.MapClaims{}
		_, _, err := jwt.NewParser().ParseUnverified(l.client.Auth.AccessJwt, claims)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing auth token",
				"Could not parse auth token: "+err.Error(),
			)
			return
		}

		// if claims map contains "scope" key
		if _, ok := claims["scope"]; ok && claims["scope"] == "com.atproto.appPass" {
			resp.Diagnostics.AddError(
				"User password required for account create/delete",
				"App password cannot be used for account creation or deletion, use user password instead.",
			)
		}
	}

	var plan accountResourceModel
	if !req.Plan.Raw.IsNull() {
		diags := req.Plan.Get(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}

		// warn if a plaintext password will be generated during account creation
		if req.State.Raw.IsNull() && plan.InitialPassword.ValueString() == "" {
			resp.Diagnostics.AddWarning(
				"Initial password not specified",
				"Initial password for account "+plan.Handle.ValueString()+" was not specified, one will be generated and included in the Terraform output in plaintext.",
			)
		}
	}
}
