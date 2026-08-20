# Community Hub Minimum

The Events MVP uses one community identity across public discovery and the
organizer portal.

## Public information architecture

1. Community identity: name, template/type, location, cover/logo.
2. Trust context: description and follower count.
3. Relationship CTA: follow/unfollow for an authenticated user; login CTA for a
   visitor.
4. Activity: published events with a valid tenant assignment appear on the
   community profile.
5. Monetization: membership tiers belong to the separate Membership domain and
   must not be inferred from event ticket prices.

## Account views

- `Komunitas Saya` lists communities where the user has an operational team
  membership.
- `Komunitas Diikuti` lists the user's consumer relationships.

These lists intentionally remain separate: following a community never grants
portal permissions, and receiving a team role does not implicitly opt the user
into community communications.

## Organizer portal

- The portal reads and updates the same tenant profile used by public
  discovery. Slug and community template/type remain immutable in this flow.
- Profile updates require `community.manage`; team changes require
  `member.manage`.
- The owner role is immutable through normal member management. A member cannot
  change or remove their own role.
- An admin can manage lower roles but cannot grant the admin role or manage
  another admin. Owner and platform admin retain that authority.
