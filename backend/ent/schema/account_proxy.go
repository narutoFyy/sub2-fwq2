package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// AccountProxy stores an account's optional multi-proxy routing configuration.
// Legacy accounts continue to use accounts.proxy_id when this table is empty.
type AccountProxy struct {
	ent.Schema
}

func (AccountProxy) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "account_proxies"},
		field.ID("account_id", "proxy_id"),
	}
}

func (AccountProxy) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (AccountProxy) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.Int64("proxy_id"),
		field.Int("concurrency").Default(1),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(0),
	}
}

func (AccountProxy) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).Unique().Required().Field("account_id"),
		edge.To("proxy", Proxy.Type).Unique().Required().Field("proxy_id"),
	}
}
