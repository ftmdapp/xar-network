/*
Package CSDT manages the storage of Collateralized Stable Debt Tokens. It handles their creation, modification, and stores the global state of all CSDTs.

Notes
 - sdk.Int is used for all the number types to maintain compatibility with internal type of sdk.Coin - saves type conversion when doing maths.
   Also it allows for changes to a CSDT to be expressed as a +ve or -ve number.
 - Only allowing one CSDT per account-collateralDenom pair for now to keep things simple.
 - Genesis forces the global debt to start at zero, ie no stable coins in existence. This could be changed.
 - The csdt module fulfills the bank keeper interface and keeps track of the liquidator module's coins. This won't be needed with module accounts.
 - GetCSDTs does not return an iterator, but instead reads out (potentially) all CSDTs from the store. This isn't a huge performance concern as it is never used during a block, only for querying.
   An iterator could be created, following the queue style construct in gov and auction, where CSDT IDs are stored under ordered keys.
   These keys could be a collateral-denom:collateral-ratio so that it is efficient to obtain the undercollateralized CSDT for a given price and liquidation ratio.
   However creating a byte sortable representation of a collateral ratio wasn't very easy so the simpler approach was chosen.

TODO
 - A shorter name for an under-collateralized CSDT would shorten a lot of function names
 - remove fake bank keeper and setup a proper liquidator module account
 - what happens if a collateral type is removed from the list of allowed ones?
 - Should the values used to generate a key for a stored struct be in the struct?
 - Add constants for the module and route names
 - Many more TODOs in the code
 - add more aggressive test cases
 - tags
 - custom error types, codespace

*/
package types
