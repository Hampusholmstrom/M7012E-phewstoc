from kodipydent import Kodi

my_kodi = Kodi('130.240.231.106')

#note = my_kodi.GUI.ShowNotification(title="Wazzup", message = "bitches")
#my_kodi.Profiles.GetCurrentProfile()
#home = my_kodi.GUI.ActivateWindow(window="home")

def loadProfiles(name):
    my_kodi.Profiles.LoadProfile(profile=name)
loadProfiles("Hampus")


# |---Profiles
# |   |
# |   |---GetCurrentProfile([, username, password, properties])
# |   |       Retrieve the current profile
# |   |
# |   |---GetProfiles([, username, password, properties, limits, sort])
# |   |       Retrieve all profiles
# |   |
# |   |---LoadProfile(profile[, username, password, prompt])
# |   |       Load the specified profile
#string profile Profile name
#[ boolean prompt = false ] Prompt for password
#[ Profiles.Password password ]
