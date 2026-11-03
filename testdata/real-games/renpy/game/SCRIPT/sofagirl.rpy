default sofa_girl = renpy.random.choice("0123")

init python:
    # клик по тянке на диване
    def sofagirl_click(transition=dissolve):
        s = sofa_girl

        while s == sofa_girl: s = renpy.random.choice("0123")

        store.sofa_girl = s

        if transition: renpy.transition(transition)

        renpy.restart_interaction()

    SofaGirlClick = renpy.curry(sofagirl_click)

default sofagirl = False

# девочка на диване
screen sofagirl(zoom=1):
    layer "master"

    if sofagirl:
        imagebutton idle "bg item sofagirl %s" % sofa_girl:
            focus_mask True
            action SofaGirlClick()
            align (.5, .5)
            at pnc_hover_at, pnc_zoom(zoom)
