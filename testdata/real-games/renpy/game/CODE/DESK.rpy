label desk:
    menu(screen="map_map"):
        "click":
            jump events

        "return":
            jump kabinet

label events:
    menu(screen="map_events"):
        "click":
            call call_event from _call_call_event

        "return":
            jump desk

    jump events