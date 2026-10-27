# Patches to the test-bed's dependencies

`build.sh` applies these to the pinned checkouts before it compiles. One that
no longer applies fails the build: upstream moved, so rebase it or drop it.

tf2-archipelago carries the same patches under `deploy/patches/`, because it
compiles this mod against its own checkouts of the same projects.

| Patch | Why |
| --- | --- |
| `cbasenpc/0001-address-is-a-cell-in-this-extension.patch` | SourceMod 1.13 makes `Address` an int64, and a plugin compiled there passes one to a native by reference. CBaseNPC's natives and callbacks read and push a 32-bit cell (`PawnAddressToPtr(params[n])`, `PushCell(PtrToPawnAddress(...))`), so every `Address` its includes declare is `any`, which is what crosses. The deprecated `CAnimationOverlay` does its arithmetic in that cell. Upstream has no 1.13 work (TF2-DMB/CBaseNPC#77, closed not planned). |
| `actions/0001-offset-a-cell-not-an-address.patch` | `BehaviorAction.Get` and `Set` cast the methodmap to `Address`, which 1.13 refuses: a methodmap is a cell and `Address` is an int64. The sum is taken in the cell and widened when it is passed. |
| `tf2utils/0001-narrow-an-offset-read-as-an-address.patch` | An offset read through the plugin's `GameConfGetAddressOffset` goes into an `any`, and 1.13 refuses the implicit int64 narrowing. The value came from `GameConfGetOffset`, an int, so `view_as<int>` loses nothing. |
| `tf2attributes/0001-gamedata-for-tf2-11138802.patch` | The pinned commit is FlaminSarge/tf2attributes#71 from June, before TF2 11087207 and 11138802 moved the two Windows signatures. These are upstream's (`747a80b`, `eb4006a8`), with the PR's `windows64` lines kept. Linux reads symbols and needs neither. |
