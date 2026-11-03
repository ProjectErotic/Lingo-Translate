label kabinet:
    if has_off_events(2, 2) == 3:
        $ sofagirl = True

    if has_off_events(2, 2) == 3:
        jump demo_end

    scene bg kabinet night
    show screen sofagirl(zoom=.275)
    $ FF()

    menu(m=1):
        "desk":
            jump desk

        "dot":
            jump sofa

        "left":
            jump mirror

        "right":
            jump bedroom

    jump kabinet
