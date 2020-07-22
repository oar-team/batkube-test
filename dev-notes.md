# Development notes

* Watching resource state is done in a very basic way, only using
    `event.Reason` only. It is sufficient for now but not robust enough with
    multiple resource types.
* Jobs finishing state is assumes to be COMPLETED_SUCCESSFULLY
* Because using events timestamps results in negative values in the csv,
    timestamps are mesured upon reception of the event in this script resulting
    in a slight delay compared to the actual values. This is not an issue
    considering the (often) very long simulation times.
